package tool

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/charlieegan3/music/pkg/tool/handlers"
	"github.com/charlieegan3/music/pkg/tool/jobs"
	"github.com/charlieegan3/toolbelt/pkg/apis"
	"github.com/gorilla/mux"
)

//go:embed migrations
var migrations embed.FS

// Music is a tool that syncs last.fm plays to bigquery
type Music struct {
	db     *sql.DB
	config *gabs.Container
}

func (m *Music) Name() string {
	return "music"
}

func (m *Music) FeatureSet() apis.FeatureSet {
	return apis.FeatureSet{
		HTTP:     true,
		HTTPHost: true,
		Config:   true,
		Jobs:     true,
		Database: true,
	}
}

func (m *Music) DatabaseMigrations() (*embed.FS, string, error) {
	return &migrations, "migrations", nil
}

func (m *Music) DatabaseSet(db *sql.DB) {
	m.db = db
}

func (m *Music) SetConfig(config map[string]any) error {
	m.config = gabs.Wrap(config)

	return nil
}

func (m *Music) Jobs() ([]apis.Job, error) {
	var j []apis.Job
	var path string
	var ok bool

	path = "jobs.sync.schedule"
	schedule, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}

	// load lastfm config
	path = "lastfm.api_key"
	lastFMAPIKey, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "lastfm.username"
	lastFMUsername, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}

	path = "spotify.access_token"
	spotifyAccessToken, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "spotify.refresh_token"
	spotifyRefreshToken, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "spotify.client_id"
	spotifyClientID, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}

	path = "spotify.client_secret"
	spotifyClientSecret, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}

	// load google config (bq & storage)
	path = "bigquery.project_id"
	projectID, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "bigquery.dataset"
	dataset, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "bigquery.table"
	table, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)

	}
	path = "google.json"
	googleJSON, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "google.covers_bucket"
	coversBucketName, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}

	return []apis.Job{
		&jobs.LastFMSync{
			ScheduleOverride:      schedule,
			APIKey:                lastFMAPIKey,
			Username:              lastFMUsername,
			GoogleCredentialsJSON: googleJSON,
			ProjectID:             projectID,
			DatasetName:           dataset,
			TableName:             table,
		},

		&jobs.SpotifySync{
			SpotifyAccessToken:  spotifyAccessToken,
			SpotifyRefreshToken: spotifyRefreshToken,
			SpotifyClientID:     spotifyClientID,
			SpotifyClientSecret: spotifyClientSecret,

			ScheduleOverride:      schedule,
			GoogleCredentialsJSON: googleJSON,
			ProjectID:             projectID,
			DatasetName:           dataset,
			TableName:             table,
		},

		&jobs.CoversSync{
			DB:                    m.db,
			ScheduleOverride:      schedule,
			GoogleCredentialsJSON: googleJSON,
			ProjectID:             projectID,
			DatasetName:           dataset,
			TableName:             table,
		},

		&jobs.CoversStore{
			DB:               m.db,
			ScheduleOverride: schedule,
			LastFMAPIKey:     lastFMAPIKey,

			GoogleCredentialsJSON: googleJSON,
			GoogleBucketName:      coversBucketName,
		},
	}, nil
}

func (m *Music) HTTPAttach(router *mux.Router) error {
	path := "bigquery.project_id"
	projectID, ok := m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "bigquery.dataset"
	dataset, ok := m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "bigquery.table"
	table, ok := m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "google.json"
	googleJSON, ok := m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "google.covers_bucket"
	coversBucketName, ok := m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}

	router.HandleFunc(
		"/",
		handlers.BuildIndexHandler(projectID, dataset, table, googleJSON),
	).Methods("GET")

	router.HandleFunc(
		"/artworks/{artist}/{album}.jpg",
		handlers.BuildArtworkHandler(coversBucketName),
	).Methods("GET")

	router.HandleFunc(
		"/{.*}",
		handlers.BuildStaticHandler(),
	).Methods("GET")

	return nil
}
func (m *Music) HTTPHost() string {
	path := "web.host"
	host, ok := m.config.Path(path).Data().(string)
	if !ok {
		return "example.com"
	}
	return host
}
func (m *Music) HTTPPath() string { return "" }

func (m *Music) ExternalJobsFuncSet(f func(job apis.ExternalJob) error) {}
