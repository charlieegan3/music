package monitoring

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/charlieegan3/music/internal/pkg/config"
)

// LatestProbe checks the last play is within the last day
func LatestProbe(cfg config.Config) error {
	URL := fmt.Sprintf("https://storage.googleapis.com/%s/stats-recent.json", cfg.Google.BucketSummary)

	res, err := http.Get(URL)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}

	jsonParsed, err := gabs.ParseJSON(data)
	if err != nil {
		return err
	}

	value, ok := jsonParsed.Path("RecentPlays.0.Timestamp").Data().(string)
	if !ok {
		return fmt.Errorf("failed to parse most recent time")
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return err
	}

	if time.Since(t).Hours() > 24 {
		return fmt.Errorf("%v hours since last play", time.Since(t).Hours())
	}

	return nil
}
