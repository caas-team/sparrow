package targets

import (
	"context"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	gitlabmock "github.com/caas-team/sparrow/pkg/sparrow/gitlab/test"
)

func Test_gitlabTargetManager_refreshTargets(t *testing.T) {
	now := time.Now()
	tooOld := now.Add(-time.Hour * 2)

	tests := []struct {
		name            string
		mockTargets     []checks.GlobalTarget
		expectedHealthy []checks.GlobalTarget
		wantErr         bool
	}{
		{
			name:            "success with 0 targets",
			mockTargets:     []checks.GlobalTarget{},
			expectedHealthy: []checks.GlobalTarget{},
		},
		{
			name: "success with 1 healthy target",
			mockTargets: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
			},
			expectedHealthy: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
			},
		},
		{
			name: "success with 1 unhealthy target",
			mockTargets: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: tooOld,
				},
			},
		},
		{
			name: "success with 1 healthy and 1 unhealthy targets",
			mockTargets: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
				{
					Url:      "test2",
					LastSeen: tooOld,
				},
			},
			expectedHealthy: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
			},
		},
		{
			name:            "failure getting targets",
			mockTargets:     nil,
			expectedHealthy: nil,
			wantErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gtm := &gitlabTargetManager{
				targets:            nil,
				gitlab:             gitlabmock.New(tt.mockTargets, tt.wantErr),
				name:               "test",
				unhealthyThreshold: time.Hour,
			}
			if err := gtm.refreshTargets(context.Background()); (err != nil) != tt.wantErr {
				t.Fatalf("refreshTargets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
