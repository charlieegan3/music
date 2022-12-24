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

	schedule string

	lastFMAPIKey   string
	lastFMUsername string

	spotifyAccessToken  string
	spotifyRefreshToken string
	spotifyClientID     string
	spotifyClientSecret string

	projectID        string
	dataset          string
	table            string
	googleJSON       string
	coversBucketName string
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

	var path string
	var ok bool

	path = "jobs.sync.schedule"
	m.schedule, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}

	// load lastfm config
	path = "lastfm.api_key"
	m.lastFMAPIKey, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "lastfm.username"
	m.lastFMUsername, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}

	path = "spotify.access_token"
	m.spotifyAccessToken, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "spotify.refresh_token"
	m.spotifyRefreshToken, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "spotify.client_id"
	m.spotifyClientID, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}

	path = "spotify.client_secret"
	m.spotifyClientSecret, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}

	// load google config (bq & storage)
	path = "bigquery.project_id"
	m.projectID, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "bigquery.dataset"
	m.dataset, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "bigquery.table"
	m.table, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "google.json"
	m.googleJSON, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "google.covers_bucket"
	m.coversBucketName, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}

	return nil
}

func (m *Music) Jobs() ([]apis.Job, error) {
	return []apis.Job{
		&jobs.LastFMSync{
			ScheduleOverride:      m.schedule,
			APIKey:                m.lastFMAPIKey,
			Username:              m.lastFMUsername,
			GoogleCredentialsJSON: m.googleJSON,
			ProjectID:             m.projectID,
			DatasetName:           m.dataset,
			TableName:             m.table,
		},

		&jobs.SpotifySync{
			SpotifyAccessToken:  m.spotifyAccessToken,
			SpotifyRefreshToken: m.spotifyRefreshToken,
			SpotifyClientID:     m.spotifyClientID,
			SpotifyClientSecret: m.spotifyClientSecret,

			ScheduleOverride:      m.schedule,
			GoogleCredentialsJSON: m.googleJSON,
			ProjectID:             m.projectID,
			DatasetName:           m.dataset,
			TableName:             m.table,
		},

		&jobs.CoversSync{
			DB:                    m.db,
			ScheduleOverride:      m.schedule,
			GoogleCredentialsJSON: m.googleJSON,
			ProjectID:             m.projectID,
			DatasetName:           m.dataset,
			TableName:             m.table,
		},

		&jobs.CoversStore{
			DB:               m.db,
			ScheduleOverride: m.schedule,
			LastFMAPIKey:     m.lastFMAPIKey,

			GoogleCredentialsJSON: m.googleJSON,
			GoogleBucketName:      m.coversBucketName,
		},

		&jobs.ArtistsIndex{
			DB:               m.db,
			ScheduleOverride: m.schedule,

			GoogleCredentialsJSON: m.googleJSON,
			ProjectID:             m.projectID,
			DatasetName:           m.dataset,
			TableName:             m.table,
		},
	}, nil
}

func (m *Music) HTTPAttach(router *mux.Router) error {
	router.HandleFunc(
		"/",
		handlers.BuildRecentHandler(m.projectID, m.dataset, m.table, m.googleJSON),
	).Methods("GET")

	router.HandleFunc(
		"/artworks/{artist}/{album}.jpg",
		handlers.BuildArtworkHandler(m.coversBucketName),
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
