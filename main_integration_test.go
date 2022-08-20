//go:build integration
// +build integration

package main

import (
	"fmt"
	"testing"
)

func TestPublishSingleSpec(t *testing.T) {
	rev = "0.0.1"

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

	err := publishMultipleSpecs(vargs)
	if err != nil {
		fmt.Println(err)
	}
}
