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
)

type API struct {
	// Spec is the path to the Open API spec file we wish to publish.
	Spec string `json:"spec"`
	// Team is the team name to publish the spec under.
	Team string `json:"team"`
	// Key is the API key for access to the spec uploader.
	Key string `json:"key"`
	// UploaderURL points to the service currently accepting spec file publishes.
	UploaderURL string `json:"uploader_url"`
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
	workspace := ""

	// Check what drone version we're running on
	if os.Getenv("DRONE_WORKSPACE") == "" { // 0.4
		err := configFromStdin(&vargs, &workspace)
		if err != nil {
			return err
		}
	} else { // 0.5+
		err := configFromEnv(&vargs, &workspace)
		if err != nil {
			return err
		}
	}

	err := validateVargs(vargs)
	if err != nil {
		return err
	}

	// Trim whitespace, to forgive the vagaries of YAML parsing.
	vargs.Key = strings.TrimSpace(vargs.Key)

	// point to file in workspace
	vargs.Spec = filepath.Join(workspace, vargs.Spec)

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

	// grabbing body in case we need to retry
	payload := body.Bytes()
	contentType := w.FormDataContentType()
	var success bool
	// make request with timeouts & retries
	for attempt := 1; attempt < 4; attempt++ {
		fmt.Printf("attempting to publish spec file: %s\n", vargs.Spec)
		r, err := http.NewRequest(http.MethodPost, vargs.UploaderURL+"?key="+vargs.Key,
			bytes.NewBuffer(payload))
		if err != nil {
			return errors.Wrap(err, "unable to create request")
		}
		r.Header.Set("Content-Type", contentType)
		resp, err := makeRequest(r)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			success = true
			break
		}

		if err != nil {
			fmt.Printf("problems publishing spec on attempt %d: %s\nsleeping for 1s\n", attempt, err)
		} else {
			respBod, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Printf("problems publishing spec on attempt %d: uploader returned with bad status of %d - %s\nsleeping for 1s\n",
				attempt, resp.StatusCode, string(respBod))
		}
		if attempt < 3 {
			time.Sleep(1 * time.Second)
		}
	}
	if success {
		fmt.Printf("successfully published spec file: %s\n", vargs.Spec)
		return nil
	}
	return errors.New("unable to publish specs after 3 attempts")
}

func makeRequest(r *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	return http.DefaultClient.Do(r)
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

func configFromStdin(vargs *API, workspace *string) error {
	// https://godoc.org/github.com/drone/drone-plugin-go/plugin
	workspaceInfo := plugin.Workspace{}
	plugin.Param("workspace", &workspaceInfo)
	plugin.Param("vargs", vargs)
	// Note this hangs if no cli args or input on STDIN
	plugin.MustParse()
	*workspace = workspaceInfo.Path
	return nil
}

func configFromEnv(vargs *API, workspace *string) error {
	// drone plugin input format du jour:
	// http://readme.drone.io/plugins/plugin-parameters/
	vargs.Spec = os.Getenv("PLUGIN_SPEC")
	vargs.Team = os.Getenv("PLUGIN_TEAM")
	vargs.Key = os.Getenv("OPENAPI_API_KEY")
	vargs.UploaderURL = os.Getenv("PLUGIN_UPLOADER_URL")
	*workspace = os.Getenv("DRONE_WORKSPACE")
	return nil
}

func validateVargs(vargs API) error {
	if vargs.Key == "" {
		return fmt.Errorf("missing required param: key")
	}
	if vargs.Spec == "" {
		return fmt.Errorf("missing required param: spec")
	}
	if vargs.Team == "" {
		return fmt.Errorf("missing required param: team")
	}
	if vargs.UploaderURL == "" {
		return fmt.Errorf("missing required param: uploader_url")
	}
	return nil
}
