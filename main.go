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

	"github.com/drone/drone-plugin-go/plugin"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
)

type API struct {
	// Spec is the path to the Open API spec file we wish to publish.
	Spec string `json:"spec"`
	// Directory is the path to the directory where Open API spec file will be picked up.
	Directory string `json:"specs_dir"`
	// Team is the team name to publish the spec under.
	Team string `json:"team"`
	// GoogleCredentials can be used to authorize spec uploads. This can optionally be
	// used instead of an API key.
	GoogleCredentials string `json:"google_credentials"`
	// Key is the API key for access to the spec uploader.
	Key string `json:"key"`
	// UploaderURL points to the service currently accepting spec file publishes.
	UploaderURL string `json:"uploader_url"`

	workspace string
}

var (
	rev string
)

func main() {
	err := wrapMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func wrapMain() error {
	if rev == "" {
		rev = "[unknown]"
	}

	fmt.Printf("Drone Open API Plugin built from %s\n", rev)

	vargs := API{}

	// Check what drone version we're running on
	if os.Getenv("DRONE_WORKSPACE") == "" { // 0.4
		err := configFromStdin(&vargs)
		if err != nil {
			return err
		}
	} else { // 0.5+
		err := configFromEnv(&vargs)
		if err != nil {
			return err
		}
	}

	err := validateVargs(vargs)
	if err != nil {
		return err
	}

	if len(vargs.Directory) > 0 {
		err = publishMultipleSpecs(vargs)
	} else {
		fmt.Printf("processing spec file: %s\n", vargs.Spec)
		err = publishSingleSpec(vargs, vargs.Spec)
	}

	return err
}

func publishMultipleSpecs(vargs API) error {
	fmt.Printf("processing multiple specs in directory: %s\n", vargs.Directory)
	files, err := ioutil.ReadDir(vargs.workspace + vargs.Directory)
	if err != nil {
		return err
	}

	for _, file := range files {
		fmt.Printf("processing spec file: %s\n", file.Name())
		err = publishSingleSpec(vargs, file.Name())
		if err != nil {
			fmt.Printf("not all files were uploaded, encountered an error for %s: %s\n", file.Name(), err)
		}
		time.Sleep(time.Second)
	}
	return err
}

func publishSingleSpec(vargs API, fileName string) error {
	var err error
	// Trim whitespace, to forgive the vagaries of YAML parsing.
	vargs.Key = strings.TrimSpace(vargs.Key)

	// point to file in workspace
	vargs.Spec = filepath.Join(vargs.workspace, vargs.Directory, fileName)

	// check spec ext to see if we need to convert YAML => JSON
	if ext := filepath.Ext(vargs.Spec); ext == ".yaml" || ext == ".yml" {
		vargs.Spec, err = convertToJSON(vargs.Spec)
		if err != nil {
			return err
		}
	}

	// post the file with timeout + retry
	return publishSpec(vargs)
}

func publishSpec(vargs API) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	// add file to request
	fw, err := w.CreateFormFile("file", filepath.Base(vargs.Spec))
	if err != nil {
		return errors.Wrap(err, "unable to init multipart form file")
	}
	spec, err := os.Open(vargs.Spec)
	if err != nil {
		return errors.Wrap(err, "unable to open spec file")
	}
	defer spec.Close()

	_, err = io.Copy(fw, spec)
	if err != nil {
		return errors.Wrap(err, "unable to write multipart form")
	}

	// add team name
	err = w.WriteField("team", vargs.Team)
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
		fmt.Printf("attempting to publish spec file: %s\n", vargs.Spec)
		status, resp, err := makeRequest(
			vargs.UploaderURL, vargs.Key, vargs.GoogleCredentials,
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

var makeRequest = func(url, key, creds, contentType string, payload []byte) (int, []byte, error) {
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

func configFromStdin(vargs *API) error {
	// https://godoc.org/github.com/drone/drone-plugin-go/plugin
	workspaceInfo := plugin.Workspace{}
	plugin.Param("workspace", &workspaceInfo)
	plugin.Param("vargs", vargs)
	// Note this hangs if no cli args or input on STDIN
	plugin.MustParse()
	vargs.workspace = workspaceInfo.Path
	return nil
}

func configFromEnv(vargs *API) error {
	// drone plugin input format du jour:
	// http://readme.drone.io/plugins/plugin-parameters/
	vargs.Spec = os.Getenv("PLUGIN_SPEC")
	vargs.Directory = os.Getenv("PLUGIN_SPECS_DIR")
	vargs.Team = os.Getenv("PLUGIN_TEAM")
	vargs.Key = os.Getenv("OPENAPI_API_KEY")
	vargs.GoogleCredentials = os.Getenv("GOOGLE_CREDENTIALS")
	vargs.UploaderURL = os.Getenv("PLUGIN_UPLOADER_URL")
	vargs.workspace = os.Getenv("DRONE_WORKSPACE")
	return nil
}

func validateVargs(vargs API) error {
	if vargs.Key == "" && vargs.GoogleCredentials == "" {
		return fmt.Errorf("missing required params: key or google_credentials")
	}
	if vargs.Directory == "" && vargs.Spec == "" {
		return fmt.Errorf("either spec or specs_dir is required")
	}
	if vargs.Directory != "" && vargs.Spec != "" {
		return fmt.Errorf("only one of spec or specs_dir was expected")
	}
	if vargs.Team == "" {
		return fmt.Errorf("missing required param: team")
	}
	if vargs.UploaderURL == "" {
		return fmt.Errorf("missing required param: uploader_url")
	}
	return nil
}
