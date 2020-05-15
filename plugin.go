package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
)

type (
	Config struct {
		// plugin-specific parameters and secrets
		Specs    []string
		Team     string
		Uploader string
		Key      string
		Token    string
	}

	Plugin struct {
		Config Config
	}
)

func (p Plugin) Validate() error {
	// plugin logic goes here
	// Validate the config
	if len(p.Config.Specs) == 0 {
		return errors.New("missing specs")
	}

	if p.Config.Team == "" {
		return errors.New("missing team")
	}

	if p.Config.Uploader == "" {
		return errors.New("missing uploader URL")
	}

	if p.Config.Key == "" && p.Config.Token == "" {
		return errors.New("missing uploader credentials")
	}

	return nil
}

func (p Plugin) Exec() error {
	// plugin logic goes here
	files, err := parseSpecs(p.Config.Specs)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !validateFileType(file) {
			fmt.Printf("not a YAML file, skipping: %v\n", file)
			continue
		}

		json, err := convertToJSON(file)
		if err != nil {
			return err
		}

		err = p.publishSpec(file, json)
	}

	return nil
}

// TODO: refactor
func (p Plugin) publishSpec(file, json string) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	// add file to request
	fw, err := w.CreateFormFile("file", filepath.Base(file))
	if err != nil {
		return errors.Wrap(err, "unable to init multipart form file")
	}
	spec, err := os.Open(file)
	if err != nil {
		return errors.Wrap(err, "unable to open spec file")
	}
	defer spec.Close()

	_, err = io.Copy(fw, spec)
	if err != nil {
		return errors.Wrap(err, "unable to write multipart form")
	}

	// add team name
	err = w.WriteField("team", p.Config.Team)
	if err != nil {
		return errors.Wrap(err, "unable to init multipart team field")
	}

	err = w.Close()
	if err != nil {
		return errors.Wrap(err, "unable to close multipart payload")
	}

	var success bool
	payload := body.Bytes()
	contentType := w.FormDataContentType()
	// make request with timeouts & retries
	for attempt := 1; attempt < 4; attempt++ {
		fmt.Printf("attempting to publish spec file: %s\n", file)
		// TODO: refactor makeRequest to a plugin type function
		status, resp, err := makeRequest(
			p.Config.Uploader, p.Config.Key, p.Config.Token,
			contentType,
			payload)
		if err == nil && status == http.StatusOK {
			success = true
			break
		}
		if err != nil {
			fmt.Printf("problems publishing spec on attempt %d: %s\n", attempt, err)
		} else {
			fmt.Printf("problems publishing spec on attempt %d: %d - %s\n",
				attempt, status, resp)
		}
		if attempt < 3 {
			dur := time.Duration(attempt) * time.Second
			fmt.Printf("sleeping for %s\n", dur)
			time.Sleep(dur)
		}
	}
	if success {
		fmt.Println("successfully published spec file")
		return nil
	}
	return errors.New("unable to publish specs after 3 attempts")

}

// TODO: refactor
func makeRequest(url, key, creds, contentType string, payload []byte) (int, []byte, error) {
	if key != "" {
		url += "?key=" + key
	}
	r, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return 0, nil, errors.Wrap(err, "unable to create request")
	}
	r.Header.Set("Content-Type", contentType)

	hc := http.DefaultClient
	if creds != "" {
		cfg, err := google.JWTConfigFromJSON([]byte(creds))
		if err != nil {
			return 0, nil, errors.Wrap(err, "unable to get JWT config from GCP creds")
		}
		cfg.PrivateClaims = map[string]interface{}{
			"target_audience": r.URL.Scheme + "://" + r.URL.Host}
		cfg.UseIDToken = true
		hc = cfg.Client(context.Background())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	resp, err := hc.Do(r)
	if err != nil {
		return 0, nil, errors.Wrap(err, "unable to make publish request")
	}
	defer resp.Body.Close()

	bod, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "unable to read publish response")
	}
	return resp.StatusCode, bod, nil
}

// TODO: refactor
func convertToJSON(pth string) (string, error) {
	// read file
	dat, err := ioutil.ReadFile(pth)
	if err != nil {
		return "", errors.Wrap(err, "unable to read spec file")
	}

	out, err := yaml.YAMLToJSON(dat)
	if err != nil {
		return "", errors.Wrap(err, "unable to convert spec file to JSON")
	}

	outName := strings.Replace(pth, filepath.Ext(pth), ".json", 1)
	err = ioutil.WriteFile(outName, out, os.ModePerm)
	return outName, errors.Wrap(err, "unable to write spec file")
}

func validateFileType(file string) bool {
	ext := filepath.Ext(file)
	return (ext == ".yaml" || ext == ".yml")
}

func parseSpecs(specs []string) ([]string, error) {
	output := []string{}

	for _, file := range specs {
		files, err := filepath.Glob(file)
		if err != nil {
			return output, err
		}
		output = append(output, files...)
	}

	return output, nil
}
