package musicRepo

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type MusicElement struct {
	MusicID       string    `json:"music_id"`
	Title         string    `json:"title"`
	Artists       []string  `json:"artists"`
	Genre         []string  `json:"genre"`
	WaveAudioLink string    `json:"wave_audio_link"`
	ArtworkLink   string    `json:"artwork_link"`
	CreatedAt     time.Time `json:"created_at"`
}

type InsertMusicElement struct {
	Title         string   `json:"title"`
	Artists       []string `json:"artists"`
	Genre         []string `json:"genre"`
	WaveAudioLink string   `json:"wave_audio_link"`
	ArtworkLink   string   `json:"artwork_link"`
}

// validate trims and checks the fields required to insert a music record.
// Mirrors puzzlesRepo.Puzzle.validate()'s pattern: reject empty/malformed
// input here rather than letting a zero-value record reach Mongo.
func (m *InsertMusicElement) validate() error {
	m.Title = strings.TrimSpace(m.Title)
	if m.Title == "" {
		return fmt.Errorf("title must not be empty")
	}

	if _, err := url.ParseRequestURI(m.WaveAudioLink); err != nil {
		return fmt.Errorf("wave_audio_link is not a valid URL: %w", err)
	}
	if _, err := url.ParseRequestURI(m.ArtworkLink); err != nil {
		return fmt.Errorf("artwork_link is not a valid URL: %w", err)
	}

	return nil
}

// normalizedTitle is what the unique index is actually built on - trimmed
// and lowercased so "Bohemian Rhapsody" and "bohemian   rhapsody" collide
// as the same track instead of silently coexisting as two documents.
func normalizedTitle(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(title), " "))
}

type internalMusicObject struct {
	ID              bson.ObjectID `bson:"_id,omitempty"`
	Title           string        `bson:"title"`
	NormalizedTitle string        `bson:"normalized_title"`
	Artists         []string      `bson:"artists"`
	Genre           []string      `bson:"genre"`
	WaveAudioLink   string        `bson:"wave_audio_link"`
	ArtworkLink     string        `bson:"artwork_link"`
	CreatedAt       time.Time     `bson:"created_at"`
}
