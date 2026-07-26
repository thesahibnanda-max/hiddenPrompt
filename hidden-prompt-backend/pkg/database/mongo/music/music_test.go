package musicRepo_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"hidden-prompt-backend/pkg/database"
	musicRepo "hidden-prompt-backend/pkg/database/mongo/music"

	"github.com/stretchr/testify/require"
)

// uniqueTitle avoids collisions across test runs/tests, since the music
// collection has no per-test cleanup hook (out of scope here - only
// GetAllMusicDetails/GetMusicByID/InsertMusicDetails exist on purpose).
func uniqueTitle(t *testing.T, suffix string) string {
	t.Helper()
	return fmt.Sprintf("TEST_MUSIC_%d_%s", time.Now().UnixNano(), suffix)
}

func insertElement(title, waveAudioLink, artworkLink string) musicRepo.InsertMusicElement {
	return musicRepo.InsertMusicElement{
		Title:         title,
		Artists:       []string{"Test Artist"},
		Genre:         []string{"Test Genre"},
		WaveAudioLink: waveAudioLink,
		ArtworkLink:   artworkLink,
	}
}

func validInsertRequest(title string) []musicRepo.InsertMusicElement {
	return []musicRepo.InsertMusicElement{insertElement(title, "https://example.com/a.wav", "https://example.com/a.png")}
}

func Test_InsertMusicDetails_RejectsInvalidInput(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)
	ctx := context.Background()

	_, err = db.MusicRepository.InsertMusicDetails(ctx, []musicRepo.InsertMusicElement{
		insertElement("", "https://example.com/a.wav", "https://example.com/a.png"),
	})
	require.Error(t, err, "empty title must be rejected")

	_, err = db.MusicRepository.InsertMusicDetails(ctx, []musicRepo.InsertMusicElement{
		insertElement(uniqueTitle(t, "badurl"), "not-a-url", "https://example.com/a.png"),
	})
	require.Error(t, err, "invalid wave_audio_link must be rejected")
}

func Test_InsertMusicDetails_SkipsExistingTitles(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)

	title := uniqueTitle(t, "dup")
	ctx := context.Background()

	first, err := db.MusicRepository.InsertMusicDetails(ctx, validInsertRequest(title))
	require.NoError(t, err)
	require.Len(t, first, 1)

	second, err := db.MusicRepository.InsertMusicDetails(ctx, validInsertRequest(title))
	require.NoError(t, err)
	require.Empty(t, second, "inserting the same title again must be skipped, not duplicated")
}

func Test_InsertMusicDetails_ConcurrentDuplicateInsertsProduceExactlyOne(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)

	title := uniqueTitle(t, "race")
	ctx := context.Background()

	const goroutines = 8
	var wg sync.WaitGroup
	insertedCounts := make([]int, goroutines)
	for i := range goroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, err := db.MusicRepository.InsertMusicDetails(ctx, validInsertRequest(title))
			require.NoError(t, err)
			insertedCounts[idx] = len(result)
		}(i)
	}
	wg.Wait()

	total := 0
	for _, c := range insertedCounts {
		total += c
	}
	require.Equal(t, 1, total, "exactly one goroutine should have won the insert race; the unique index must reject the rest")
}

func Test_GetMusicByID(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)

	ctx := context.Background()
	title := uniqueTitle(t, "byid")
	inserted, err := db.MusicRepository.InsertMusicDetails(ctx, validInsertRequest(title))
	require.NoError(t, err)
	require.Len(t, inserted, 1)

	got, err := db.MusicRepository.GetMusicByID(ctx, inserted[0].MusicID)
	require.NoError(t, err)
	require.Equal(t, title, got.Title)
	require.Equal(t, inserted[0].MusicID, got.MusicID)
}

func Test_GetMusicByID_NotFound(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)

	_, err = db.MusicRepository.GetMusicByID(context.Background(), "aaaaaaaaaaaaaaaaaaaaaaaa")
	require.Error(t, err)
}

func Test_GetAllMusicDetails_RespectsLimit(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)

	ctx := context.Background()
	base := uniqueTitle(t, "list")
	for i := range 3 {
		_, err := db.MusicRepository.InsertMusicDetails(ctx, validInsertRequest(fmt.Sprintf("%s_%d", base, i)))
		require.NoError(t, err)
	}

	page, err := db.MusicRepository.GetAllMusicDetails(ctx, 1, 0)
	require.NoError(t, err)
	require.Len(t, page, 1, "limit=1 must return exactly one record, not the whole collection")
}
