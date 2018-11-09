package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
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
	video, err := fetchDataForVideo("awX9XkPG5oY")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(video.ID)
	fmt.Println(video.Track)
	fmt.Println(video.Artist)
	fmt.Println(video.Album)
	fmt.Println(video.Artwork)
	fmt.Println(video.Duration)
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

	r := regexp.MustCompile(`PT((?P<Hours>\d+)H)?((?P<Minutes>\d+)M)?(?P<Seconds>\d+)S`)

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
