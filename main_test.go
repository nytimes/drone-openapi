//go:build unit
// +build unit

package main

import (
	"fmt"
	"testing"
)

func TestValidateArgs(t *testing.T) {
	testCases := []struct {
		desc        string
		vargs       API
		expectedErr string
	}{
		{
			desc:        "key_or_google_creds_expected",
			vargs:       API{},
			expectedErr: "missing required params: key or google_credentials",
		},
		{
			desc:        "spec_or_specs_dir_expected",
			vargs:       API{Key: "1", GoogleCredentials: "creds"},
			expectedErr: "either spec or specs_dir is required",
		},
		{
			desc:        "both_spec_and_specs_dir__not_expected",
			vargs:       API{Key: "1", GoogleCredentials: "creds", Spec: "spec", Directory: "dir"},
			expectedErr: "only one of spec or specs_dir was expected",
		},
		{
			desc:        "team_is_expected",
			vargs:       API{Key: "1", GoogleCredentials: "creds", Spec: "spec"},
			expectedErr: "missing required param: team",
		},
		{
			desc:        "uploader_url_is_expected",
			vargs:       API{Key: "1", GoogleCredentials: "creds", Spec: "spec", Team: "team"},
			expectedErr: "missing required param: uploader_url",
		},
		{
			desc:        "good_request_1",
			vargs:       API{Key: "1", GoogleCredentials: "creds", Spec: "spec", Team: "team", UploaderURL: "url"},
			expectedErr: "",
		},
		{
			desc:        "good_request_2",
			vargs:       API{Key: "1", GoogleCredentials: "creds", Directory: "spec_dir", Team: "team", UploaderURL: "url"},
			expectedErr: "",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			err := validateVargs(tC.vargs)

			if tC.expectedErr == "" {
				if err != nil {
					t.Fatalf("Error not expected. got: `%v`", err)
				}
			} else if tC.expectedErr != err.Error() {
				t.Fatalf("Error not in the expected format: expected: `%v`, got: `%v`", tC.expectedErr, err.Error())
			}
		})
	}
}

func TestPublishSingleSpec(t *testing.T) {
	rev = "0.0.1"

	makeRequest = func(url, key, creds, contentType string, payload []byte) (int, []byte, error) {
		return 200, []byte(""), nil
	}

	vargs := API{
		workspace:         "./",
		Key:               "expected_in_env",
		GoogleCredentials: "expected_in_env",
		Spec:              "testdata/specs/hello_world_spec.yml",
		Team:              "drone-openapi",
		UploaderURL:       "https://apis.nyt.net/update",
	}

	err := publishSpec(vargs)
	if err != nil {
		fmt.Println(err)
	}
}

func TestPublishMultipleSpecs(t *testing.T) {
	rev = "0.0.1"

	vargs := API{
		workspace:         "./",
		Key:               "expected_in_env",
		GoogleCredentials: "expected_in_env",
		Directory:         "testdata/specs",
		Team:              "drone-openapi",
		UploaderURL:       "https://apis.nyt.net/update",
	}

	makeRequest = func(url, key, creds, contentType string, payload []byte) (int, []byte, error) {
		return 200, []byte(""), nil
	}

	err := publishMultipleSpecs(vargs)
	if err != nil {
		fmt.Println(err)
	}
}
