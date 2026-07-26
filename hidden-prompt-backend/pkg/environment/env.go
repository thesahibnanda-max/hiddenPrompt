package env

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func SetDotEnv(folderPath string) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		slog.Error("Failed to read env directory",
			slog.String("path", folderPath),
			slog.Any("error", err),
		)
		return
	}
	slog.Info("Enteries", slog.Any("values", entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Optional: only load .env files
		if filepath.Ext(entry.Name()) != ".env" {
			continue
		}

		path := filepath.Join(folderPath, entry.Name())

		b, err := os.ReadFile(path)
		if err != nil {
			slog.Error("Failed to read env file",
				slog.String("file", path),
				slog.Any("error", err),
			)
			continue
		}

		envMap, err := godotenv.Unmarshal(string(b))
		if err != nil {
			slog.Error("Failed to parse env file",
				slog.String("file", path),
				slog.Any("error", err),
			)
			continue
		}

		for key, value := range envMap {
			if err := os.Setenv(key, value); err != nil {
				slog.Error("Failed to set environment variable",
					slog.String("key", key),
					slog.Any("error", err),
				)
				continue
			}

			slog.Info("Loaded environment variable",
				slog.String("key", key),
			)
		}
	}
}
