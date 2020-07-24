package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/structs"
	"github.com/hashicorp/go-multierror"
)

// Config stores all application config
type Config struct {
	Google struct {
		BucketBackup   string `yaml:"bucket_backup"`
		BucketSummary  string `yaml:"bucket_summary"`
		Dataset        string `yaml:"dataset"`
		Project        string `yaml:"project"`
		SvcAccountJSON string `yaml:"svc_account_json"`
		Table          string `yaml:"table"`
		TableEnrich    string `yaml:"table_enrich"`
		TableTransfer  string `yaml:"table_transfer"`
	} `yaml:"google"`
	Shazam struct {
		Cookie   string `yaml:"cookie"`
		Referrer string `yaml:"referrer"`
		URL      string `yaml:"url"`
	} `yaml:"shazam"`
	Soundcloud struct {
		Oauth string `yaml:"oauth"`
		URL   string `yaml:"url"`
	} `yaml:"soundcloud"`
	Spotify struct {
		AccessToken  string `yaml:"access_token"`
		AuthState    string `yaml:"auth_state"`
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		RefreshToken string `yaml:"refresh_token"`
	} `yaml:"spotify"`
	Youtube struct {
		AccessToken  string `yaml:"access_token"`
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		Cookie       string `yaml:"cookie"`
		RefreshToken string `yaml:"refresh_token"`
	} `yaml:"youtube"`
}

// ValidateAndInit validates config and sets the Google credentials in
// the environment. It returns the cleanup function to the caller.
func (c *Config) ValidateAndInit() func() {
	var result *multierror.Error

	for k, v := range structs.Map(c) {
		subConfig, ok := v.(map[string]interface{})
		if !ok {
			log.Fatalf("%T was not %T", v, subConfig)
		}
		for sk, sv := range subConfig {
			value, ok := sv.(string)
			if !ok {
				log.Fatalf("%T was not %T", sv, value)
			}
			if value == "" {
				result = multierror.Append(result, fmt.Errorf("missing %s.%s", k, sk))
			}
		}
	}

	if result.ErrorOrNil() != nil {
		log.Fatalf("failed, missing config: %v", result)
	}

	tmpfile, err := ioutil.TempFile("", "google.*.json")
	if err != nil {
		log.Fatal(err)
	}
	content := []byte(c.Google.SvcAccountJSON)
	if _, err := tmpfile.Write(content); err != nil {
		tmpfile.Close()
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tmpfile.Name())

	return func() {
		defer os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
		if err := os.Remove(tmpfile.Name()); err != nil {
			log.Fatal(err)
		}
	}
}
