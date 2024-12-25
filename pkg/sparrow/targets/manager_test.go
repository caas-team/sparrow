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

	remotemock "github.com/caas-team/sparrow/pkg/sparrow/targets/remote/test"
)

const (
	testCheckInterval        = 100 * time.Millisecond
	testRegistrationInterval = 150 * time.Millisecond
	testUpdateInterval       = 150 * time.Millisecond
)

// Test_gitlabTargetManager_refreshTargets tests that the refreshTargets method
// will fetch the targets from the remote instance and update the local
// targets list. When an unhealthyTheshold is set, it will also unregister
// unhealthy targets
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
			remote := remotemock.New(tt.mockTargets)
			if tt.wantErr != nil {
				remote.SetFetchFilesErr(tt.wantErr)
			}
			gtm := &manager{
				targets:    nil,
				interactor: remote,
				name:       "test",
				cfg:        General{UnhealthyThreshold: time.Hour, Scheme: "https"},
				metrics:    newMetrics(),
			}
			if err := gtm.refreshTargets(context.Background()); (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("refreshTargets() error = %v, wantErr %v", err, tt.wantErr)
			}

			if gtm.registered != tt.expectedRegisteredAfter {
				t.Fatalf("expected registered to be %v, got %v", tt.expectedRegisteredAfter, gtm.registered)
			}
		})
	}
}

// Test_gitlabTargetManager_refreshTargets_No_Threshold tests that the
// refreshTargets method will not unregister unhealthy targets if the
// unhealthyThreshold is 0
func Test_gitlabTargetManager_refreshTargets_No_Threshold(t *testing.T) {
	tests := []struct {
		name                    string
		mockTargets             []checks.GlobalTarget
		expectedHealthy         []checks.GlobalTarget
		expectedRegisteredAfter bool
		wantErr                 error
	}{
		{
			name: "success with 1 target",
			mockTargets: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: time.Now().Add(-time.Hour * 24),
				},
			},
			expectedHealthy: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: time.Now().Add(-time.Hour * 2),
				},
			},
			expectedRegisteredAfter: true,
		},
		{
			name: "success with 2 old targets",
			mockTargets: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: time.Now().Add(-time.Hour * 24),
				},
				{
					Url:      "https://test2",
					LastSeen: time.Now().Add(-time.Hour * 24),
				},
			},
			expectedHealthy: []checks.GlobalTarget{
				{
					Url:      "https://test",
					LastSeen: time.Now().Add(-time.Hour * 24),
				},
				{
					Url:      "https://test2",
					LastSeen: time.Now().Add(-time.Hour * 24),
				},
			},
			expectedRegisteredAfter: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote := remotemock.New(tt.mockTargets)
			gtm := &manager{
				targets:    nil,
				interactor: remote,
				name:       "test",
				cfg:        General{UnhealthyThreshold: 0, Scheme: "https"},
				metrics:    newMetrics(),
			}
			if err := gtm.refreshTargets(context.Background()); (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("refreshTargets() error = %v, wantErr %v", err, tt.wantErr)
			}

			if gtm.registered != tt.expectedRegisteredAfter {
				t.Fatalf("expected registered to be %v, got %v", tt.expectedRegisteredAfter, gtm.registered)
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
			gtm := &manager{
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

// Test_gitlabTargetManager_registerSparrow tests that the register method will
// register the sparrow instance in the remote instance
func Test_gitlabTargetManager_register(t *testing.T) {
	tests := []struct {
		name       string
		wantErr    bool
		wantPutErr bool
	}{
		{
			name: "success",
		},
		{
			name:       "failure - failed to register",
			wantErr:    true,
			wantPutErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glmock := remotemock.New(nil)
			if tt.wantPutErr {
				glmock.SetPostFileErr(fmt.Errorf("failed to register"))
			}
			gtm := &manager{
				interactor: glmock,
				metrics:    newMetrics(),
			}
			if err := gtm.register(context.Background()); (err != nil) != tt.wantErr {
				t.Fatalf("register() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if !gtm.registered {
					t.Fatalf("register() did not register the instance")
				}
			}
		})
	}
}

// Test_gitlabTargetManager_update tests that the update
// method will update the registration of the sparrow instance in the remote instance
func Test_gitlabTargetManager_update(t *testing.T) {
	tests := []struct {
		name         string
		wantPutError bool
	}{
		{
			name: "success - update registration",
		},
		{
			name:         "failure - failed to update registration",
			wantPutError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glmock := remotemock.New(nil)
			if tt.wantPutError {
				glmock.SetPutFileErr(fmt.Errorf("failed to update registration"))
			}
			gtm := &manager{
				interactor: glmock,
				registered: true,
			}
			wantErr := tt.wantPutError
			if err := gtm.update(context.Background()); (err != nil) != wantErr {
				t.Fatalf("update() error = %v, wantErr %v", err, wantErr)
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

	testTarget := "https://some.target"
	glmock := remotemock.New(
		[]checks.GlobalTarget{
			{
				Url:      testTarget,
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

			time.Sleep(testUpdateInterval * 2)
			if gtm.GetTargets()[0].Url != testTarget {
				t.Fatalf("Reconcile() did not receive the correct target")
			}

			gtm.mu.Lock()
			if !gtm.registered {
				t.Fatalf("Reconcile() did not register")
			}
			gtm.mu.Unlock()

			err := gtm.Shutdown(ctx)
			if err != nil {
				t.Fatalf("Reconcile() failed to shutdown")
			}
		})
	}
}

// Test_gitlabTargetManager_Reconcile_Registration_Update tests that the Reconcile
// method will register the sparrow, and then update the registration after the
// registration interval has passed
func Test_gitlabTargetManager_Reconcile_Registration_Update(t *testing.T) {
	glmock := remotemock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://some.sparrow",
				LastSeen: time.Now(),
			},
		},
	)

	gtm := mockGitlabTargetManager(glmock, "test")
	gtm.cfg.RegistrationInterval = 10 * time.Millisecond
	gtm.cfg.UpdateInterval = 100 * time.Millisecond

	ctx := context.Background()
	go func() {
		err := gtm.Reconcile(ctx)
		if err != nil {
			t.Error("Reconcile() should not have returned an error")
			return
		}
	}()

	timeout := time.After(gtm.cfg.UpdateInterval * 3)
	select {
	case <-timeout:
		gtm.mu.Lock()
		if !gtm.registered {
			t.Fatalf("Reconcile() should be registered")
		}
		gtm.mu.Unlock()

		// check that the post call was made once, to create the registration
		if glmock.PostFileCount() != 1 {
			t.Fatalf("Reconcile() should have registered the instance once")
		}

		// check that the put call was made twice, to update the registration
		if glmock.PutFileCount() != 2 {
			t.Fatalf("Reconcile() should have updated the registration twice")
		}

		gtm.mu.Lock()
		if !gtm.registered {
			t.Fatalf("Reconcile() should have registered the sparrow")
		}
		gtm.mu.Unlock()

		err := gtm.Shutdown(ctx)
		if err != nil {
			t.Fatalf("Reconcile() failed to shutdown")
		}
	default:
	}

	t.Logf("Reconcile() successfully registered and updated the sparrow")
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
					Url:      "https://some.sparrow",
					LastSeen: time.Now(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glmock := remotemock.New(tt.targets)

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
			if tt.postErr != nil && gtm.registered {
				t.Fatalf("Reconcile() should not have registered")
			}

			if tt.putError != nil && !gtm.registered {
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
	glmock := remotemock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://some.sparrow",
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
	if !gtm.registered {
		t.Fatalf("Reconcile() should still be registered")
	}
	gtm.mu.Unlock()

	// assert mock calls
	if !glmock.PostFileCalled() || !glmock.PutFileCalled() {
		t.Fatalf("Reconcile() should have made calls to the gitlab API")
	}
}

// Test_gitlabTargetManager_Reconcile_Context_Done tests that the Reconcile
// method will shut down gracefully when the context is done.
func Test_gitlabTargetManager_Reconcile_Context_Done(t *testing.T) {
	glmock := remotemock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://some.sparrow",
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
	if gtm.registered {
		t.Fatalf("Reconcile() should not be registered")
	}
	gtm.mu.Unlock()

	// assert mock calls
	if glmock.PostFileCalled() || glmock.PutFileCalled() {
		t.Fatalf("Reconcile() should not have made calls to the gitlab API")
	}
}

// Test_gitlabTargetManager_Reconcile_Shutdown tests that the Reconcile
// method will shut down gracefully when the Shutdown method is called.
func Test_gitlabTargetManager_Reconcile_Shutdown(t *testing.T) {
	glmock := remotemock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://some.sparrow",
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
	if gtm.registered {
		t.Fatalf("Reconcile() should not be registered")
	}
	gtm.mu.Unlock()

	// assert mock calls
	if !glmock.PostFileCalled() || !glmock.PutFileCalled() {
		t.Fatalf("Reconcile() should have made calls to the gitlab API")
	}
}

// Test_gitlabTargetManager_Reconcile_Shutdown_Fail_Unregister tests that the Reconcile
// method will fail the graceful shutdown when the Shutdown method is called
// and the unregistering fails.
func Test_gitlabTargetManager_Reconcile_Shutdown_Fail_Unregister(t *testing.T) {
	glmock := remotemock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://some.sparrow",
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
	if !gtm.registered {
		t.Fatalf("Reconcile() should still be registered")
	}
	gtm.mu.Unlock()

	// assert mock calls
	if !glmock.PostFileCalled() || !glmock.PutFileCalled() {
		t.Fatalf("Reconcile() should have made calls to the gitlab API")
	}
}

// Test_gitlabTargetManager_Reconcile_No_Registration tests that the Reconcile
// method will not register the instance if the registration interval is 0
func Test_gitlabTargetManager_Reconcile_No_Registration(t *testing.T) {
	glmock := remotemock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://some.sparrow",
				LastSeen: time.Now(),
			},
		},
	)

	gtm := mockGitlabTargetManager(glmock, "test")
	gtm.cfg.RegistrationInterval = 0

	ctx := context.Background()
	go func() {
		err := gtm.Reconcile(ctx)
		if err != nil {
			t.Error("Reconcile() should not have returned an error")
			return
		}
	}()

	time.Sleep(time.Millisecond * 250)

	gtm.mu.Lock()
	if gtm.registered {
		t.Fatalf("Reconcile() should not be registered")
	}
	gtm.mu.Unlock()

	// check that no calls were made
	if glmock.PostFileCalled() || glmock.PutFileCalled() {
		t.Fatalf("Reconcile() should not have registered the instance")
	}
}

// Test_gitlabTargetManager_Reconcile_No_Update tests that the Reconcile
// method will not update the registration if the update interval is 0
func Test_gitlabTargetManager_Reconcile_No_Update(t *testing.T) {
	glmock := remotemock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://some.sparrow",
				LastSeen: time.Now(),
			},
		},
	)

	gtm := mockGitlabTargetManager(glmock, "test")
	gtm.cfg.UpdateInterval = 0

	ctx := context.Background()
	go func() {
		err := gtm.Reconcile(ctx)
		if err != nil {
			t.Error("Reconcile() should not have returned an error")
			return
		}
	}()

	time.Sleep(time.Millisecond * 250)

	gtm.mu.Lock()
	if !gtm.registered {
		t.Fatalf("Reconcile() should be registered")
	}
	gtm.mu.Unlock()

	// assert mock calls
	if glmock.PutFileCalled() {
		t.Fatalf("Reconcile() should not have updated the registration")
	}
	if glmock.PostFileCount() != 1 {
		t.Fatalf("Reconcile() should have registered the instance")
	}
}

// Test_gitlabTargetManager_Reconcile_No_Registration_No_Update tests that the Reconcile
// method will not register the instance if the registration interval is 0
// and will not update the registration if the update interval is 0
func Test_gitlabTargetManager_Reconcile_No_Registration_No_Update(t *testing.T) {
	glmock := remotemock.New(
		[]checks.GlobalTarget{
			{
				Url:      "https://some.sparrow",
				LastSeen: time.Now(),
			},
		},
	)

	gtm := mockGitlabTargetManager(glmock, "test")
	gtm.cfg.RegistrationInterval = 0
	gtm.cfg.UpdateInterval = 0

	ctx := context.Background()
	go func() {
		err := gtm.Reconcile(ctx)
		if err != nil {
			t.Error("Reconcile() should not have returned an error")
			return
		}
	}()

	time.Sleep(time.Millisecond * 250)

	gtm.mu.Lock()
	if gtm.registered {
		t.Fatalf("Reconcile() should not be registered")
	}
	gtm.mu.Unlock()

	// assert mock calls
	if glmock.PostFileCalled() {
		t.Fatalf("Reconcile() should not have registered the instance")
	}
	if glmock.PutFileCalled() {
		t.Fatalf("Reconcile() should not have updated the registration")
	}
}

func mockGitlabTargetManager(g *remotemock.MockClient, name string) *manager { //nolint: unparam // irrelevant
	return &manager{
		targets:    nil,
		mu:         sync.RWMutex{},
		done:       make(chan struct{}, 1),
		interactor: g,
		name:       name,
		metrics:    newMetrics(),
		cfg: General{
			CheckInterval:        100 * time.Millisecond,
			UnhealthyThreshold:   1 * time.Second,
			RegistrationInterval: testRegistrationInterval,
			UpdateInterval:       testUpdateInterval,
		},
		registered: false,
	}
}
