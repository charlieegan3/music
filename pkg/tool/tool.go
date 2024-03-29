package tool

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/Jeffail/gabs/v2"
	"github.com/gorilla/mux"

	"github.com/charlieegan3/music/pkg/tool/cache"
	"github.com/charlieegan3/music/pkg/tool/handlers"
	"github.com/charlieegan3/music/pkg/tool/jobs"
	"github.com/charlieegan3/toolbelt/pkg/apis"
)

//go:embed migrations
var migrations embed.FS

// Music is a tool that syncs last.fm plays to bigquery
type Music struct {
	db     *sql.DB
	config *gabs.Container

	lastFMschedule  string
	spotifySchedule string
	coversSchedule  string
	artistsSchedule string
	backupSchedule  string

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
	backupBucketName string
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

	path = "jobs.lastfm.schedule"
	m.lastFMschedule, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "jobs.spotify.schedule"
	m.spotifySchedule, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "jobs.covers.schedule"
	m.coversSchedule, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "jobs.artists.schedule"
	m.artistsSchedule, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}
	path = "jobs.backup.schedule"
	m.backupSchedule, ok = m.config.Path(path).Data().(string)
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
	path = "google.backup_bucket"
	m.backupBucketName, ok = m.config.Path(path).Data().(string)
	if !ok {
		return fmt.Errorf("missing required config path: %s", path)
	}

	return nil
}

func (m *Music) Jobs() ([]apis.Job, error) {
	return []apis.Job{
		&jobs.LastFMSync{
			ScheduleOverride:      m.lastFMschedule,
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

			ScheduleOverride:      m.spotifySchedule,
			GoogleCredentialsJSON: m.googleJSON,
			ProjectID:             m.projectID,
			DatasetName:           m.dataset,
			TableName:             m.table,
		},

		&jobs.CoversSync{
			DB:                    m.db,
			ScheduleOverride:      m.coversSchedule,
			GoogleCredentialsJSON: m.googleJSON,
			ProjectID:             m.projectID,
			DatasetName:           m.dataset,
			TableName:             m.table,
		},

		&jobs.CoversStore{
			DB:               m.db,
			ScheduleOverride: m.coversSchedule,
			LastFMAPIKey:     m.lastFMAPIKey,

			GoogleCredentialsJSON: m.googleJSON,
			GoogleBucketName:      m.coversBucketName,
		},

		&jobs.BuildIndex{
			DB:               m.db,
			ScheduleOverride: m.artistsSchedule,

			GoogleCredentialsJSON: m.googleJSON,
			ProjectID:             m.projectID,
			DatasetName:           m.dataset,
			TableName:             m.table,
		},

		&jobs.Backup{
			DB:               m.db,
			ScheduleOverride: m.backupSchedule,

			GoogleCredentialsJSON: m.googleJSON,
			ProjectID:             m.projectID,
			DatasetName:           m.dataset,
			TableName:             m.table,
			BackupBucketName:      m.backupBucketName,
		},
	}, nil
}

func (m *Music) HTTPAttach(router *mux.Router) error {

	store := cache.NewStorage()

	router.HandleFunc(
		"/menu",
		handlers.BuildMenuHandler(),
	).Methods("GET")

	router.Handle(
		"/",
		cache.Middleware(
			"24h",
			store,
			handlers.BuildTopHandler(m.projectID, m.dataset, m.table, m.googleJSON),
		),
	).Methods("GET")

	router.Handle(
		"/recent{format:.*}",
		cache.Middleware(
			"15m",
			store,
			handlers.BuildRecentHandler(m.projectID, m.dataset, m.table, m.googleJSON),
		),
	).Methods("GET")

	router.Handle(
		"/months",
		cache.Middleware(
			"168h",
			store,
			handlers.BuildMonthsHandler(m.projectID, m.dataset, m.table, m.googleJSON),
		),
	).Methods("GET")

	router.Handle(
		"/artists",
		cache.Middleware(
			"24h",
			store,
			handlers.BuildArtistSearchHandler(m.projectID, m.dataset, m.table, m.googleJSON),
		),
	).Methods("GET")

	router.Handle(
		"/artists/{artistSlug}",
		cache.Middleware(
			"24h",
			store,
			handlers.BuildArtistHandler(m.db, m.projectID, m.dataset, m.table, m.googleJSON),
		),
	).Methods("GET")

	router.Handle(
		"/artists/{artistSlug}/albums/{albumSlug}",
		cache.Middleware(
			"24h",
			store,
			handlers.BuildArtistAlbumHandler(m.db, m.projectID, m.dataset, m.table, m.googleJSON),
		),
	).Methods("GET")

	router.Handle(
		"/artists/{artistSlug}/tracks/{trackSlug}",
		cache.Middleware(
			"24h",
			store,
			handlers.BuildArtistTrackHandler(m.db, m.projectID, m.dataset, m.table, m.googleJSON),
		),
	).Methods("GET")

	router.Handle(
		"/artists/{artistSlug}/albums/{albumSlug}/tracks/{trackSlug}",
		cache.Middleware(
			"24h",
			store,
			handlers.BuildArtistAlbumTrackHandler(m.db, m.projectID, m.dataset, m.table, m.googleJSON),
		),
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
