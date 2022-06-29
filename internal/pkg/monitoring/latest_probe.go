package monitoring

import (
	"bytes"
	"encoding/json"
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
		datab := map[string]string{
			"Title": "Play Data Warning",
			"Body":  fmt.Sprintf("%.0f hours since last play", time.Since(t).Hours()),
			"URL":   "",
		}

		b, err := json.Marshal(datab)
		if err != nil {
			return fmt.Errorf("failed to marshal webhook body: %w", err)
		}

		client := &http.Client{}
		req, err := http.NewRequest("POST", cfg.Webhook.Endpoint, bytes.NewBuffer(b))

		req.Header.Add("Content-Type", "application/json; charset=utf-8")

		_, err = client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send webhook: %w", err)
		}
	}

	return nil
}
