package models

import (
	"time"

	"github.com/samber/lo"
)

type MusicDetail struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Artists       []string  `json:"artists"`
	Genre         []string  `json:"genre"`
	WaveAudioLink string    `json:"wave_audio_link"`
	ArtworkLink   string    `json:"artwork_link"`
	CreatedAt     time.Time `json:"created_at"`
}

func (m *MusicDetail) Clone() MusicDetail {
	if m == nil {
		return MusicDetail{}
	}

	return MusicDetail{
		ID:            m.ID,
		Title:         m.Title,
		Artists:       lo.Slice(m.Artists, 0, len(m.Artists)),
		Genre:         lo.Slice(m.Genre, 0, len(m.Genre)),
		WaveAudioLink: m.WaveAudioLink,
		ArtworkLink:   m.ArtworkLink,
		CreatedAt:     m.CreatedAt,
	}
}
