package config

import (
	"testing"
	"time"

	"github.com/go-test/deep"
)

func Test_NewTargetManagerConfig_Gitlab(t *testing.T) {
	got := NewTargetManagerConfig("testdata/tmconfig.yaml")
	want := TargetManagerConfig{
		CheckInterval:        300 * time.Second,
		RegistrationInterval: 600 * time.Second,
		UnhealthyThreshold:   900 * time.Second,
		Gitlab: GitlabTargetManagerConfig{
			BaseURL:   "https://gitlab.devops.telekom.de",
			ProjectID: 666,
			Token:     "gitlab-token",
		},
	}

	if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
	}
}
