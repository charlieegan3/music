package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/charlieegan3/music/internal/pkg/spotify"
	"github.com/charlieegan3/music/pkg/tool/bq"
)

// SpotifySync is a job that syncs spotify plays to bigquery
type SpotifySync struct {
	ScheduleOverride string

	SpotifyAccessToken  string
	SpotifyRefreshToken string
	SpotifyClientID     string
	SpotifyClientSecret string

	GoogleCredentialsJSON string
	ProjectID             string
	DatasetName           string
	TableName             string
}

func (s *SpotifySync) Name() string {
	return "spotify-sync"
}

func (s *SpotifySync) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	go func() {
		err := spotify.Sync(
			s.SpotifyAccessToken,
			s.SpotifyRefreshToken,
			s.SpotifyClientID,
			s.SpotifyClientSecret,
			s.GoogleCredentialsJSON,
			s.ProjectID,
			s.DatasetName,
			s.TableName,
			bq.JSONSchema,
		)
		if err != nil {
			errCh <- fmt.Errorf("failed to sync spotify: %v", err)
		}

		doneCh <- true
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-errCh:
		return fmt.Errorf("job failed with error: %s", e)
	case <-doneCh:
		return nil
	}
}

func (s *SpotifySync) Timeout() time.Duration {
	return 30 * time.Second
}

func (s *SpotifySync) Schedule() string {
	if s.ScheduleOverride != "" {
		return s.ScheduleOverride
	}
	return "0 0 6 * * *"
}
