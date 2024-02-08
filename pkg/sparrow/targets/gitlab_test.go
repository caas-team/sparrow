// sparrow
// (C) 2023, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package targets

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"

	gitlabmock "github.com/caas-team/sparrow/pkg/sparrow/gitlab/test"
)

func Test_gitlabTargetManager_refreshTargets(t *testing.T) {
	now := time.Now()
	tooOld := now.Add(-time.Hour * 2)

	tests := []struct {
		name                    string
		mockTargets             []checks.GlobalTarget
		expectedHealthy         []checks.GlobalTarget
		expectedRegisteredAfter bool
		wantErr                 error
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
					Url:      "https://test",
					LastSeen: now,
				},
			},
			expectedHealthy: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: now,
				},
			},
			expectedRegisteredAfter: true,
		},
		{
			name: "success with 1 unhealthy target",
			mockTargets: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: tooOld,
				},
			},
			expectedRegisteredAfter: true,
		},
		{
			name: "success with 1 healthy and 1 unhealthy targets",
			mockTargets: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: now,
				},
				{
					Url:      "https://test2",
					LastSeen: tooOld,
				},
			},
			expectedHealthy: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: now,
				},
			},
			expectedRegisteredAfter: true,
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
				targets: nil,
				gitlab:  gitlab,
				name:    "test",
				cfg:     cfg{unhealthyThreshold: time.Hour},
			}
			if err := gtm.refreshTargets(context.Background()); (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("refreshTargets() error = %v, wantErr %v", err, tt.wantErr)
			}

			if gtm.Registered() != tt.expectedRegisteredAfter {
				t.Fatalf("expected registered to be %v, got %v", tt.expectedRegisteredAfter, gtm.Registered())
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
					Url:      "https://test",
					LastSeen: now,
				},
			},
			want: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: now,
				},
			},
		},
		{
			name: "success with 2 targets",
			targets: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: now,
				},
				{
					Url:      "https://test2",
					LastSeen: now,
				},
			},
			want: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: now,
				},
				{
					Url:      "https://test2",
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
				Url:      "https://test",
				LastSeen: time.Now(),
			},
		},
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gtm := mockGitlabTargetManager(glmock, "test")
			ctx := context.Background()
			go func() {
				err := gtm.Reconcile(ctx)
				if err != nil {
					t.Error("Reconcile() should not have returned an error")
					return
				}
			}()

			time.Sleep(time.Millisecond * 300)
			if gtm.GetTargets()[0].Url != "https://test" {
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
		targets    []checks.GlobalTarget
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
			targets: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: time.Now(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glmock := gitlabmock.New(tt.targets)

			gtm := mockGitlabTargetManager(glmock, "test")
			glmock.SetPostFileErr(tt.postErr)
			glmock.SetPutFileErr(tt.putError)

			ctx := context.Background()
			go func() {
				err := gtm.Reconcile(ctx)
				if err != nil {
					t.Error("Reconcile() should not have returned an error")
					return
				}
			}()

			time.Sleep(time.Millisecond * 300)

			gtm.mu.Lock()
			if tt.postErr != nil && gtm.Registered() {
				t.Fatalf("Reconcile() should not have registered")
			}

			if tt.putError != nil && !gtm.Registered() {
				t.Fatalf("Reconcile() should still be registered")
			}
			gtm.mu.Unlock()

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
				Url:      "https://test",
				LastSeen: time.Now(),
			},
		},
	)

	gtm := mockGitlabTargetManager(glmock, "test")

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		err := gtm.Reconcile(ctx)
		if err == nil {
			t.Error("Reconcile() should have returned an error")
			return
		}
	}()

	time.Sleep(time.Millisecond * 250)
	cancel()
	time.Sleep(time.Millisecond * 250)

	gtm.mu.Lock()
	if !gtm.Registered() {
		t.Fatalf("Reconcile() should still be registered")
	}
	gtm.mu.Unlock()
}

// Test_gitlabTargetManager_Reconcile_Context_Done tests that the Reconcile
// method will shut down gracefully when the context is done.
func Test_gitlabTargetManager_Reconcile_Context_Done(t *testing.T) {
	glmock := gitlabmock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://test",
				LastSeen: time.Now(),
			},
		},
	)

	gtm := mockGitlabTargetManager(glmock, "test")

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	go func() {
		err := gtm.Reconcile(ctx)
		if err == nil {
			t.Error("Reconcile() should have returned an error")
			return
		}
	}()

	time.Sleep(time.Millisecond * 15)

	gtm.mu.Lock()
	if gtm.Registered() {
		t.Fatalf("Reconcile() should not be registered")
	}
	gtm.mu.Unlock()
}

// Test_gitlabTargetManager_Reconcile_Shutdown tests that the Reconcile
// method will shut down gracefully when the Shutdown method is called.
func Test_gitlabTargetManager_Reconcile_Shutdown(t *testing.T) {
	glmock := gitlabmock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://test",
				LastSeen: time.Now(),
			},
		},
	)

	gtm := mockGitlabTargetManager(glmock, "test")

	ctx := context.Background()
	go func() {
		err := gtm.Reconcile(ctx)
		if err != nil {
			t.Error("Reconcile() should not have returned an error")
			return
		}
	}()

	time.Sleep(time.Millisecond * 250)

	err := gtm.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Reconcile() failed to shutdown")
	}

	gtm.mu.Lock()
	if gtm.Registered() {
		t.Fatalf("Reconcile() should not be registered")
	}
	gtm.mu.Unlock()
}

// Test_gitlabTargetManager_Reconcile_Shutdown_Fail_Unregister tests that the Reconcile
// method will fail the graceful shutdown when the Shutdown method is called
// and the unregistering fails.
func Test_gitlabTargetManager_Reconcile_Shutdown_Fail_Unregister(t *testing.T) {
	glmock := gitlabmock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://test",
				LastSeen: time.Now(),
			},
		},
	)

	gtm := mockGitlabTargetManager(glmock, "test")
	glmock.SetDeleteFileErr(errors.New("gitlab API error"))

	ctx := context.Background()
	go func() {
		err := gtm.Reconcile(ctx)
		if err != nil {
			t.Error("Reconcile() should not have returned an error")
			return
		}
	}()

	time.Sleep(time.Millisecond * 250)

	err := gtm.Shutdown(ctx)
	if err == nil {
		t.Fatalf("Reconcile() should have failed to shutdown")
	}

	gtm.mu.Lock()
	// instance should still be registered because the unregister failed
	if !gtm.Registered() {
		t.Fatalf("Reconcile() should still be registered")
	}
	gtm.mu.Unlock()
}

func mockGitlabTargetManager(g *gitlabmock.MockClient, name string) *gitlabTargetManager { //nolint: unparam // irrelevant
	return &gitlabTargetManager{
		targets: nil,
		mu:      sync.RWMutex{},
		done:    make(chan struct{}, 1),
		gitlab:  g,
		name:    name,
		cfg: cfg{
			checkInterval:        100 * time.Millisecond,
			unhealthyThreshold:   1 * time.Second,
			registrationInterval: 150 * time.Millisecond,
		},
		registered: false,
	}
}
