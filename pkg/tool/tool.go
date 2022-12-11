package tool

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/charlieegan3/music/pkg/tool/handlers"
	"github.com/charlieegan3/toolbelt/pkg/apis"
	"github.com/gorilla/mux"
)

// Music is a tool that syncs last.fm plays to bigquery
type Music struct {
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
	}
}

func (m *Music) SetConfig(config map[string]any) error {
	m.config = gabs.Wrap(config)

	return nil
}
func (m *Music) Jobs() ([]apis.Job, error) {
	var j []apis.Job
	var path string
	var ok bool

	// load lastfm config
	path = "jobs.sync.schedule"
	schedule, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "lastfm.api_key"
	apiKey, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}
	path = "lastfm.username"
	username, ok := m.config.Path(path).Data().(string)
	if !ok {
		return j, fmt.Errorf("missing required config path: %s", path)
	}

	// load bigquery/google config
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

	return []apis.Job{
		&LastFMSync{
			ScheduleOverride:      schedule,
			APIKey:                apiKey,
			Username:              username,
			ProjectID:             projectID,
			DatasetName:           dataset,
			TableName:             table,
			GoogleCredentialsJSON: googleJSON,
		},
	}, nil
}

func (m *Music) HTTPAttach(router *mux.Router) error {
	router.HandleFunc(
		"/",
		handlers.BuildIndexHandler(),
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
func (m *Music) DatabaseMigrations() (*embed.FS, string, error) {
	return &embed.FS{}, "migrations", nil
}
func (m *Music) DatabaseSet(db *sql.DB) {}
