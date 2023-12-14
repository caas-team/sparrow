package targets

import (
	"context"
	"errors"
	"fmt"
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
		wantErr         error
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
			wantErr:         fmt.Errorf("failed to fetch files"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlab := gitlabmock.New(tt.mockTargets)
			if tt.wantErr != nil {
				gitlab.SetFetchFilesErr(tt.wantErr)
			}
			gtm := &gitlabTargetManager{
				targets:            nil,
				gitlab:             gitlab,
				name:               "test",
				unhealthyThreshold: time.Hour,
			}
			if err := gtm.refreshTargets(context.Background()); (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("refreshTargets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_gitlabTargetManager_GetTargets(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		targets []checks.GlobalTarget
		want    []checks.GlobalTarget
	}{
		{
			name:    "success with 0 targets",
			targets: nil,
			want:    nil,
		},
		{
			name: "success with 1 target",
			targets: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
			},
			want: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
			},
		},
		{
			name: "success with 2 targets",
			targets: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
				{
					Url:      "test2",
					LastSeen: now,
				},
			},
			want: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
				{
					Url:      "test2",
					LastSeen: now,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gtm := &gitlabTargetManager{
				targets: tt.targets,
			}
			got := gtm.GetTargets()

			if len(got) != len(tt.want) {
				t.Fatalf("GetTargets() got = %v, want %v", got, tt.want)
			}

			for i := range got {
				if got[i].Url != tt.want[i].Url {
					t.Fatalf("GetTargets() got = %v, want %v", got, tt.want)
				}
				if !got[i].LastSeen.Equal(tt.want[i].LastSeen) {
					t.Fatalf("GetTargets() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func Test_gitlabTargetManager_updateRegistration(t *testing.T) {
	tests := []struct {
		name          string
		registered    bool
		wantPostError bool
		wantPutError  bool
	}{
		{
			name: "success - first registration",
		},
		{
			name:       "success - update registration",
			registered: true,
		},
		{
			name:          "failure - failed to register",
			wantPostError: true,
		},
		{
			name:         "failure - failed to update registration",
			registered:   true,
			wantPutError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glmock := gitlabmock.New(nil)
			if tt.wantPostError {
				glmock.SetPostFileErr(fmt.Errorf("failed to register"))
			}
			if tt.wantPutError {
				glmock.SetPutFileErr(fmt.Errorf("failed to update registration"))
			}
			gtm := &gitlabTargetManager{
				gitlab:     glmock,
				registered: tt.registered,
			}
			wantErr := tt.wantPutError || tt.wantPostError
			if err := gtm.updateRegistration(context.Background()); (err != nil) != wantErr {
				t.Fatalf("updateRegistration() error = %v, wantErr %v", err, wantErr)
			}
		})
	}
}

// Test_gitlabTargetManager_Reconcile_success tests that the Reconcile method
// will register the target if it is not registered yet and update the
// registration if it is already registered
func Test_gitlabTargetManager_Reconcile_success(t *testing.T) {
	tests := []struct {
		name          string
		registered    bool
		wantPostError bool
		wantPutError  bool
	}{
		{
			name: "success - first registration",
		},
		{
			name:       "success - update registration",
			registered: true,
		},
	}

	glmock := gitlabmock.New(
		[]checks.GlobalTarget{
			{
				Url:      "test",
				LastSeen: time.Now(),
			},
		},
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gtm := NewGitlabManager(
				glmock,
				"test",
				time.Millisecond*100,
				time.Hour*1,
				time.Millisecond*150,
			)
			ctx := context.Background()
			go func() {
				gtm.Reconcile(ctx)
			}()

			time.Sleep(time.Millisecond * 300)
			if gtm.GetTargets()[0].Url != "test" {
				t.Fatalf("Reconcile() did not receive the correct target")
			}
			if !gtm.Registered() {
				t.Fatalf("Reconcile() did not register")
			}

			err := gtm.Shutdown(ctx)
			if err != nil {
				t.Fatalf("Reconcile() failed to shutdown")
			}
		})
	}
}

// Test_gitlabTargetManager_Reconcile_failure tests that the Reconcile method
// will handle API failures gracefully
func Test_gitlabTargetManager_Reconcile_failure(t *testing.T) {
	tests := []struct {
		name       string
		registered bool
		postErr    error
		putError   error
	}{
		{
			name:    "failure - failed to register",
			postErr: errors.New("failed to register"),
		},
		{
			name:       "failure - failed to update registration",
			registered: true,
			putError:   errors.New("failed to update registration"),
		},
	}

	glmock := gitlabmock.New(
		[]checks.GlobalTarget{
			{
				Url:      "test",
				LastSeen: time.Now(),
			},
		},
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gtm := NewGitlabManager(
				glmock,
				"test",
				time.Millisecond*100,
				time.Hour*1,
				time.Millisecond*150,
			)
			glmock.SetPostFileErr(tt.postErr)
			glmock.SetPutFileErr(tt.putError)

			ctx := context.Background()
			go func() {
				gtm.Reconcile(ctx)
			}()

			time.Sleep(time.Millisecond * 300)

			if tt.postErr != nil && gtm.Registered() {
				t.Fatalf("Reconcile() should not have registered")
			}

			if tt.putError != nil && !gtm.Registered() {
				t.Fatalf("Reconcile() should still be registered")
			}

			err := gtm.Shutdown(ctx)
			if err != nil {
				t.Fatalf("Reconcile() failed to shutdown")
			}
		})
	}
}

// Test_gitlabTargetManager_Reconcile_Context_Canceled tests that the Reconcile
// method will shutdown gracefully when the context is canceled.
func Test_gitlabTargetManager_Reconcile_Context_Canceled(t *testing.T) {
	glmock := gitlabmock.New(
		[]checks.GlobalTarget{
			{
				Url:      "test",
				LastSeen: time.Now(),
			},
		},
	)

	gtm := NewGitlabManager(
		glmock,
		"test",
		time.Millisecond*100,
		time.Hour*1,
		time.Millisecond*150,
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		gtm.Reconcile(ctx)
	}()

	time.Sleep(time.Millisecond * 250)
	cancel()
	time.Sleep(time.Millisecond * 100)

	// instance shouldn't be registered anymore
	if gtm.Registered() {
		t.Fatalf("Reconcile() should not be registered")
	}
}
