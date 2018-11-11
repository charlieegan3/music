package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/tidwall/gjson"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	youtube "google.golang.org/api/youtube/v3"
)

type metadataRowRenderer struct {
	Contents []struct {
		SimpleText string `json:"simpleText"`
	} `json:"contents"`
	Title struct {
		SimpleText string `json:"simpleText"`
	} `json:"title"`
}

// Video represents a video played on youtube, to be linked with a point in time to be used as a play/scrobble
type Video struct {
	ID         string
	Track      string
	Artist     string
	Album      string
	Artwork    string
	Duration   int
	CategoryID string
}

// Youtube downloads data from youtube
func Youtube() {
	// Creates a bq client.
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	tableName := os.Getenv("GOOGLE_TABLE")
	accountJSON := os.Getenv("GOOGLE_JSON")

	creds, err := google.CredentialsFromJSON(ctx, []byte(accountJSON), bigquery.Scope)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	bigqueryClient, err := bigquery.NewClient(ctx, projectID, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		os.Exit(1)
	}
	// loads in the table schema from file
	jsonSchema, err := ioutil.ReadFile("schema.json")
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
		os.Exit(1)
	}
	schema, err := bigquery.SchemaFromJSON(jsonSchema)
	if err != nil {
		log.Fatalf("Failed to parse schema: %v", err)
		os.Exit(1)
	}
	u := bigqueryClient.Dataset(datasetName).Table(tableName).Uploader()

	// fetch the two most recent plays that can be matched against the historic data to find a progress point
	loggedPlays := mostRecentlyLogged(ctx, bigqueryClient, projectID, datasetName, tableName)

	// get recent plays
	videoIDs, err := fetchRecentPlays()
	if err != nil {
		panic(err)
	}

	if len(videoIDs) < 1 {
		fmt.Println("no videoIDs found")
		return
	}

	cutoff := len(videoIDs)

	// find where the logged plays and new plays meet
	if len(loggedPlays) > 0 {
		for i, v := range videoIDs {
			if v == loggedPlays[0] {
				cutoff = i
				break
			}
		}
	}

	videoIDs = videoIDs[0:cutoff]
	if len(videoIDs) < 1 {
		fmt.Println("No new plays to import")
		return
	}
	fmt.Println("importing", len(videoIDs), "plays")

	var recentPlays []Video
	for _, v := range videoIDs {
		video, err := fetchDataForVideo(v)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Music or entertainment
		if video.CategoryID == "10" || video.CategoryID == "24" {
			// reverse to import in order
			recentPlays = append([]Video{video}, recentPlays...)
		} else {
			fmt.Println("skipping:", v, "(not music)")
		}
	}

	for _, video := range recentPlays {
		// creates items to be saved in bigquery
		var vss []*bigquery.ValuesSaver
		vss = append(vss, &bigquery.ValuesSaver{
			Schema:   schema,
			InsertID: fmt.Sprintf("%v%v", time.Now().UTC(), video.ID),
			Row: []bigquery.Value{
				video.Track,
				video.Artist,
				video.Album,
				time.Now().UTC(),
				video.Duration * 1000,
				"", // spotify_id
				video.Artwork,
				time.Now().UTC(),
				"youtube",        // source
				video.ID,         // youtube_id
				video.CategoryID, // youtube_category_id
				"",               // soundcloud_id
				"",               // soundcloud_permalink
			},
		})

		fmt.Println("upload", video.Track, "-", video.Artist)

		// upload the items
		err = u.Put(ctx, vss)
		if err != nil {
			if pmErr, ok := err.(bigquery.PutMultiError); ok {
				for _, rowInsertionError := range pmErr {
					log.Println(rowInsertionError.Errors)
				}
				return
			}

			log.Println(err)
		}

		// to make sure that the play timestamps are correctly ordered
		time.Sleep(500 * time.Millisecond)
	}
}

func mostRecentlyLogged(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) []string {
	queryString := fmt.Sprintf(
		"SELECT youtube_id as ID FROM `%s.%s.%s` WHERE youtube_id IS NOT NULL AND youtube_id != \"\" ORDER BY timestamp DESC LIMIT 20",
		projectID,
		datasetName,
		tableName,
	)

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var videos []string
	for {
		var v Video
		err := it.Next(&v)
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		videos = append(videos, v.ID)
	}

	return videos
}

func fetchRecentPlays() ([]string, error) {
	var videoIDs []string

	req, err := http.NewRequest("GET", "https://www.youtube.com/feed/history", nil)
	if err != nil {
		return videoIDs, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:62.0) Gecko/20100101 Firefox/62.0")
	req.Header.Set("Cookie", os.Getenv("YOUTUBE_COOKIE"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return videoIDs, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return videoIDs, err
	}

	videoIDRegex := regexp.MustCompile(`videoId":"([^"]+)"`)

	for _, v := range videoIDRegex.FindAllSubmatch(body, -1) {
		existing := false
		for _, v2 := range videoIDs {
			if string(v[1]) == v2 {
				existing = true
			}
		}

		if existing == false {
			videoIDs = append(videoIDs, string(v[1]))
		}
	}

	return videoIDs, nil
}

func fetchDataForVideo(videoID string) (Video, error) {
	var result Video
	client := getClient()
	service, err := youtube.New(client)
	call := service.Videos.List("snippet,contentDetails,statistics")
	call.Id(videoID)
	resp, err := call.Do()
	if err != nil {
		panic(err)
	}
	if len(resp.Items) < 1 {
		return result, errors.New("failed to find video for ID" + videoID)
	}

	video := resp.Items[0]

	var metadata map[string]string
	completedMetadata, err := extractMetadata(videoID)
	if err == nil {
		metadata = completedMetadata
	}

	title := video.Snippet.Title

	guessedMetadata := guessMetadata(title)

	result.ID = videoID
	result.Duration = parse8601Duration(video.ContentDetails.Duration)
	result.Artwork = video.Snippet.Thumbnails.Default.Url
	result.CategoryID = video.Snippet.CategoryId

	if len(guessedMetadata) > 1 {
		result.Artist = guessedMetadata[0]
	} else {
		result.Artist = video.Snippet.ChannelTitle
	}
	if len(guessedMetadata) > 1 {
		result.Track = guessedMetadata[1]
	} else {
		result.Track = guessedMetadata[0]
	}

	if metadata["Song"] != "" {
		if strings.Contains(strings.ToLower(title), strings.ToLower(metadata["Song"])) {
			result.Track = metadata["Song"]
		}
	}
	if metadata["Artist"] != "" {
		if strings.Contains(strings.ToLower(title), strings.ToLower(metadata["Artist"])) {
			result.Artist = metadata["Artist"]
		}
	}
	if metadata["Album"] != "" {
		result.Album = metadata["Album"]
	}

	return result, nil
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient() *http.Client {
	ctx := context.Background()

	config := &oauth2.Config{
		ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
		RedirectURL:  "https://developers.google.com/oauthplayground",
		Scopes:       []string{youtube.YoutubeReadonlyScope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://www.googleapis.com/oauth2/v3/token",
		},
	}

	tok := oauth2.Token{
		AccessToken:  os.Getenv("YOUTUBE_ACCESS_TOKEN"),
		RefreshToken: os.Getenv("YOUTUBE_REFRESH_TOKEN"),
		Expiry:       time.Now().AddDate(0, 0, -1),
	}

	return config.Client(ctx, &tok)
}

func parse8601Duration(duration string) int {
	seconds := 0

	r := regexp.MustCompile(`PT((?P<Hours>\d+)H)?((?P<Minutes>\d+)M)?((?P<Seconds>\d+)S)?`)

	matches := r.FindStringSubmatch(duration)
	parts := r.SubexpNames()

	for i, val := range parts {
		switch val {
		case "Hours":
			if integer, err := strconv.ParseInt(matches[i], 10, 32); err == nil {
				seconds += int(integer) * 60 * 60
			}
		case "Minutes":
			if integer, err := strconv.ParseInt(matches[i], 10, 32); err == nil {
				seconds += int(integer) * 60
			}
		case "Seconds":
			if integer, err := strconv.ParseInt(matches[i], 10, 32); err == nil {
				seconds += int(integer)
			}
		}
	}

	return seconds
}

func guessMetadata(title string) []string {
	bars := regexp.MustCompile(`\|\S+\|`)
	video := regexp.MustCompile(`(?i)[\(\[\{].*video.*[\)\]\}]`)

	title = bars.ReplaceAllString(title, "")
	title = video.ReplaceAllString(title, "")

	parts := strings.Split(title, " - ")

	switch len(parts) {
	case 1:
		return []string{parts[0]}
	default:
		return parts[0:2]
	}
}

func extractMetadata(videoID string) (map[string]string, error) {
	metadata := make(map[string]string)
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://www.youtube.com/watch?v="+videoID, nil)
	if err != nil {
		return metadata, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:62.0) Gecko/20100101 Firefox/62.0")

	resp, err := client.Do(req)
	if err != nil {
		return metadata, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return metadata, err
	}

	r := regexp.MustCompile("ytInitialData[^\n]+")
	matches := r.FindAllString(string(body), 1)

	if len(matches) == 0 {
		return metadata, errors.New("ytInitialData not found in DOM")
	}

	line := matches[0]

	r = regexp.MustCompile("{.*}")
	matches = r.FindAllString(line, -1)

	if len(matches) == 0 {
		return metadata, errors.New("failed to extract ytInitialData json")
	}

	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(matches[0]), &m); err != nil {
		return metadata, err
	}

	for _, v := range parseMap(m, "", "metadataRowRenderer") {
		matchingObject := gjson.Get(matches[0], v[1:len(v)])
		var mr metadataRowRenderer
		if err := json.Unmarshal([]byte(matchingObject.String()), &mr); err != nil {
			continue
		}

		if len(mr.Contents) > 0 {
			metadata[mr.Title.SimpleText] = mr.Contents[0].SimpleText
		}
	}

	return metadata, nil
}

func parseMap(aMap map[string]interface{}, path string, searchKey string) []string {
	var matches []string
	for key, val := range aMap {
		newPath := path + "." + key

		if key == searchKey {
			matches = append(matches, newPath)
			continue
		}

		switch val.(type) {
		case map[string]interface{}:
			matches = append(matches, parseMap(val.(map[string]interface{}), newPath, searchKey)...)
		case []interface{}:
			matches = append(matches, parseArray(val.([]interface{}), newPath, searchKey)...)
		}
	}
	return matches
}

func parseArray(anArray []interface{}, path string, searchKey string) []string {
	var matches []string
	for i, val := range anArray {
		newPath := path + "." + strconv.Itoa(i)
		switch val.(type) {
		case map[string]interface{}:
			matches = append(matches, parseMap(val.(map[string]interface{}), newPath, searchKey)...)
		case []interface{}:
			matches = append(matches, parseArray(val.([]interface{}), newPath, searchKey)...)
		}
	}
	return matches
}
