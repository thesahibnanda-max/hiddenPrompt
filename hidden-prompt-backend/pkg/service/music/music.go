package music

import (
	"context"
	"errors"
	"fmt"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/database"
	musicRepo "hidden-prompt-backend/pkg/database/mongo/music"
	customErrors "hidden-prompt-backend/pkg/errors"
	"hidden-prompt-backend/pkg/models"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bsm/redislock"
	goxJsonUtils "github.com/devlibx/gox-base/v2/serialization/utils/json"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
)

type Service interface {

	// InsertMusicDetails
	// Inserts but returns all details for client ease
	InsertMusicDetails(ctx context.Context, musicDetails []models.MusicDetail) ([]models.MusicDetail, error)

	// GetAllMusicDetails
	// Returns all Music Details
	GetAllMusicDetails(ctx context.Context) ([]models.MusicDetail, error)
}

func NewService(db *database.DatabaseParams) (Service, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	return &service{
		ttlForL1Cache: constants.InMemoryTTLForMusicDetails,
		db:            db,
	}, nil
}

type service struct {
	l1CacheMutex    sync.Mutex
	l1Cache         []models.MusicDetail
	l1CacheWhenSave time.Time
	ttlForL1Cache   time.Duration
	db              *database.DatabaseParams
}

func (s *service) InsertMusicDetails(ctx context.Context, musicDetails []models.MusicDetail) ([]models.MusicDetail, error) {
	if len(musicDetails) == 0 {
		return nil, &customErrors.MusicDetailsRequired{Err: errors.New("nothing to save")}
	}

	musicDetailsInsertDB := lo.Map(musicDetails, func(item models.MusicDetail, _ int) musicRepo.InsertMusicElement {
		return musicRepo.InsertMusicElement{
			Title:         item.Title,
			Artists:       item.Artists,
			Genre:         item.Genre,
			WaveAudioLink: item.WaveAudioLink,
			ArtworkLink:   item.ArtworkLink,
		}
	})

	_, err := s.db.MusicRepository.InsertMusicDetails(ctx, musicDetailsInsertDB)
	if err != nil {
		return nil, err
	}

	musicDetailsDB, err := s.db.MusicRepository.GetAllMusicDetails(ctx, constants.MusicDBLimit, constants.MusicDBOffset)
	if err != nil {
		return nil, err
	}

	musicDetailsNew := s.mapAndSortMusicDetails(musicDetailsDB)
	s.cacheMusicDetails(ctx, musicDetailsNew)

	return musicDetailsNew, nil
}

func (s *service) GetAllMusicDetails(ctx context.Context) ([]models.MusicDetail, error) {

	// first get from L1 Cache
	if musicDetails, ok := s.getFromL1Cache(); ok {
		return musicDetails, nil
	}

	// try L2 cache
	if musicDetails, err := s.getFromL2Cache(ctx); err == nil {
		return musicDetails, nil
	} else if !errors.Is(err, redis.Nil) {
		slog.Error("error from getting via music details via redis", slog.Any("error", err))
	}

	// get from db
	// we have kept it this way delibrately, we don't want caller to decide limit and offset
	// as we won't ever have more than 20 music details so why overkill
	return s.getAllMusicDetailsWithLock(ctx)
}

// getAllMusicDetailsWithLock guards the DB fetch with a Redis lock to
// collapse a thundering herd of concurrent cache-miss requests into a
// single DB hit. Retries for a bounded time (RedisLockRetryBackoff *
// RedisLockMaxRetries) to wait for whichever request is already fetching -
// if that request has already populated the cache by the time the lock is
// obtained, this returns the now-warm cache instead of hitting the DB
// again. Any failure to obtain or release the lock fails open (falls back
// to fetching directly) rather than erroring the caller - this lock is a
// best-effort optimization, not a correctness requirement, so lock
// infrastructure problems must never turn into a broken endpoint.
func (s *service) getAllMusicDetailsWithLock(ctx context.Context) ([]models.MusicDetail, error) {
	lock, err := s.db.RedisLocker.Obtain(ctx, constants.RedisLockKeyForMusicDetails, constants.RedisLockTTLForMusicDetails, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(constants.RedisLockRetryBackoff), constants.RedisLockMaxRetries),
	})
	if err != nil {
		if !errors.Is(err, redislock.ErrNotObtained) {
			slog.Error("error obtaining redis lock for music details", slog.Any("error", err))
		}
		return s.fetchMusicDetailsFromDB(ctx)
	}
	defer func() {
		if releaseErr := lock.Release(context.WithoutCancel(ctx)); releaseErr != nil {
			slog.Error("error releasing redis lock for music details", slog.Any("error", releaseErr))
		}
	}()

	// Whoever held the lock before us may have already populated the
	// cache while we were waiting - check again before hitting the DB
	// ourselves, otherwise the lock only serializes DB access instead of
	// actually preventing redundant hits.
	if musicDetails, ok := s.getFromL1Cache(); ok {
		return musicDetails, nil
	}
	if musicDetails, err := s.getFromL2Cache(ctx); err == nil {
		return musicDetails, nil
	} else if !errors.Is(err, redis.Nil) {
		slog.Error("error from getting via music details via redis", slog.Any("error", err))
	}

	return s.fetchMusicDetailsFromDB(ctx)
}

func (s *service) fetchMusicDetailsFromDB(ctx context.Context) ([]models.MusicDetail, error) {
	musicDetailsDB, err := s.db.MusicRepository.GetAllMusicDetails(ctx, constants.MusicDBLimit, constants.MusicDBOffset)
	if err != nil {
		return nil, err
	}

	musicDetails := s.mapAndSortMusicDetails(musicDetailsDB)
	s.cacheMusicDetails(ctx, musicDetails)

	return musicDetails, nil
}

// mapAndSortMusicDetails is the single place that maps from the repo type
// and sorts by title - every path that populates the cache or returns to
// a caller goes through this, so cache reads (L1/L2 hits) and fresh DB
// fetches are never inconsistently ordered relative to each other.
func (s *service) mapAndSortMusicDetails(items []musicRepo.MusicElement) []models.MusicDetail {
	musicDetails := lo.Map(items, func(item musicRepo.MusicElement, _ int) models.MusicDetail {
		return models.MusicDetail{
			ID:            item.MusicID,
			Title:         item.Title,
			Artists:       item.Artists,
			Genre:         item.Genre,
			WaveAudioLink: item.WaveAudioLink,
			ArtworkLink:   item.ArtworkLink,
			CreatedAt:     item.CreatedAt,
		}
	})
	slices.SortStableFunc(musicDetails, func(a, b models.MusicDetail) int {
		return strings.Compare(a.Title, b.Title)
	})
	return musicDetails
}

// cacheMusicDetails populates L1 (sync) and L2 (async, best-effort) from
// an already-sorted slice, deep-copying for each so neither cache can be
// mutated through a shared backing array with what's returned to the
// caller.
func (s *service) cacheMusicDetails(ctx context.Context, musicDetails []models.MusicDetail) {
	go s.setInL1Cache(s.deepCopyMusicDetails(musicDetails))
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
	go func(musicDetailsInsideGoFunc []models.MusicDetail, bgCtx context.Context, cancel context.CancelFunc) {
		defer cancel()
		if setErr := s.setInL2Cache(bgCtx, musicDetailsInsideGoFunc); setErr != nil {
			slog.Error("error in setting in L2 Cache", slog.Any("error", setErr))
		}
	}(s.deepCopyMusicDetails(musicDetails), bgCtx, cancel)
}

func (s *service) setInL1Cache(musicDetails []models.MusicDetail) {
	s.l1CacheMutex.Lock()
	defer s.l1CacheMutex.Unlock()

	s.l1Cache = musicDetails
	s.l1CacheWhenSave = time.Now()
}

func (s *service) getFromL1Cache() ([]models.MusicDetail, bool) {
	s.l1CacheMutex.Lock()
	defer s.l1CacheMutex.Unlock()

	if s.l1CacheWhenSave.IsZero() || time.Since(s.l1CacheWhenSave) >= s.ttlForL1Cache {
		return nil, false
	}

	return s.l1Cache, true
}

func (s *service) setInL2Cache(ctx context.Context, musicDetails []models.MusicDetail) error {
	gottenString, err := goxJsonUtils.ObjectToString(musicDetails)
	if err != nil {
		return err
	}
	return s.db.RedisRepository.Set(ctx, constants.RedisKeyForMusicDetails, gottenString, constants.RedisTTLForMusicDetails)
}

func (s *service) getFromL2Cache(ctx context.Context) ([]models.MusicDetail, error) {

	gottenString, err := s.db.RedisRepository.Get(ctx, constants.RedisKeyForMusicDetails)
	if err != nil {
		return nil, err
	}

	musicDetails, err := goxJsonUtils.StringToObject[[]models.MusicDetail](gottenString)
	if err != nil {
		return nil, err
	}
	return musicDetails, nil
}

func (s *service) deepCopyMusicDetails(m []models.MusicDetail) []models.MusicDetail {
	return lo.Map(m, func(item models.MusicDetail, _ int) models.MusicDetail {
		return item.Clone()
	})
}
