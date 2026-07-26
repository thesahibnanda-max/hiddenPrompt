package musicRepo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrNoRows mirrors puzzlesRepo/usersRepo's sentinel - wraps
// mongo.ErrNoDocuments so callers never need to import the driver package
// just to check "not found".
var ErrNoRows = errors.New("no music rows available")

// defaultListLimit caps GetAllMusicDetails when the caller doesn't specify
// one, so a bare call can never load an unbounded collection into memory.
const defaultListLimit = 100

// uniqueTitleIndexName is the name given to the unique index created at
// construction time - referenced only for logging/debugging clarity.
const uniqueTitleIndexName = "normalized_title_unique"

type Repository interface {
	// GetAllMusicDetails returns up to limit records, skipping the first
	// offset (sorted newest-first). limit <= 0 uses defaultListLimit.
	GetAllMusicDetails(ctx context.Context, limit, offset int64) ([]MusicElement, error)
	GetMusicByID(ctx context.Context, id string) (*MusicElement, error)
	// InsertMusicDetails inserts every element in req whose (normalized)
	// title isn't already present, and returns the elements actually
	// inserted - not the whole collection. Titles that collide with an
	// existing record are skipped, not treated as an error.
	InsertMusicDetails(ctx context.Context, req []InsertMusicElement) ([]MusicElement, error)
}

// NewRepository wraps mongoConn and provisions the indexes this package
// relies on - mirrors puzzlesRepo.NewRepo/usersRepo.NewRepository applying
// their embedded schema.sql at construction time, so every repo already
// has what it needs to operate correctly before it's handed to a caller.
func NewRepository(ctx context.Context, mongoConn *mongo.Collection) (Repository, error) {
	if mongoConn == nil {
		return nil, fmt.Errorf("music: mongoConn is nil")
	}

	if _, err := mongoConn.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "normalized_title", Value: 1}},
		Options: options.Index().SetUnique(true).SetName(uniqueTitleIndexName),
	}); err != nil {
		return nil, fmt.Errorf("music: create unique title index: %w", err)
	}

	return &repo{mongoConn: mongoConn}, nil
}

type repo struct {
	mongoConn *mongo.Collection
}

func (r *repo) InsertMusicDetails(ctx context.Context, req []InsertMusicElement) ([]MusicElement, error) {
	if len(req) == 0 {
		return []MusicElement{}, nil
	}

	inserted := make([]MusicElement, 0, len(req))
	for _, item := range req {
		if err := item.validate(); err != nil {
			return nil, fmt.Errorf("music: invalid insert request: %w", err)
		}

		now := time.Now()
		id := bson.NewObjectID()
		doc := internalMusicObject{
			ID:              id,
			Title:           item.Title,
			NormalizedTitle: normalizedTitle(item.Title),
			Artists:         item.Artists,
			Genre:           item.Genre,
			WaveAudioLink:   item.WaveAudioLink,
			ArtworkLink:     item.ArtworkLink,
			CreatedAt:       now,
		}

		// One insert per title (rather than InsertMany for the whole
		// batch) so a single duplicate title doesn't fail the rest of an
		// otherwise-valid batch, and so the unique index - not an
		// application-level pre-scan - is what actually enforces
		// uniqueness, closing the race a scan-then-insert approach can't.
		if _, err := r.mongoConn.InsertOne(ctx, doc); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				slog.Info("music: skipping duplicate title", slog.String("title", item.Title))
				continue
			}
			return nil, fmt.Errorf("music: insert %q: %w", item.Title, err)
		}

		inserted = append(inserted, toMusicElement(doc))
	}

	return inserted, nil
}

func (r *repo) GetAllMusicDetails(ctx context.Context, limit, offset int64) ([]MusicElement, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}

	cursor, err := r.mongoConn.Find(ctx, bson.D{}, options.Find().
		SetLimit(limit).
		SetSkip(offset).
		SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("music: find: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			slog.Error("music: err in cursor.Close()", slog.Any("error", err))
		}
	}()

	elements := make([]MusicElement, 0)
	for cursor.Next(ctx) {
		var m internalMusicObject
		if err := cursor.Decode(&m); err != nil {
			return nil, fmt.Errorf("music: decode: %w", err)
		}
		elements = append(elements, toMusicElement(m))
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("music: cursor: %w", err)
	}

	slices.SortStableFunc(elements, func(a, b MusicElement) int {
		return strings.Compare(a.Title, b.Title)
	})

	return elements, nil
}

func (r *repo) GetMusicByID(ctx context.Context, id string) (*MusicElement, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("music: invalid id %q: %w", id, err)
	}

	var m internalMusicObject
	if err := r.mongoConn.FindOne(ctx, bson.D{{Key: "_id", Value: objectID}}).Decode(&m); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNoRows
		}
		return nil, fmt.Errorf("music: find by id: %w", err)
	}

	element := toMusicElement(m)
	return &element, nil
}

func toMusicElement(m internalMusicObject) MusicElement {
	return MusicElement{
		MusicID:       m.ID.Hex(),
		Title:         m.Title,
		Artists:       m.Artists,
		Genre:         m.Genre,
		WaveAudioLink: m.WaveAudioLink,
		ArtworkLink:   m.ArtworkLink,
		CreatedAt:     m.CreatedAt,
	}
}
