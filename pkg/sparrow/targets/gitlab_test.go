package targets

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
)

func Test_gitlab_fetchFileList(t *testing.T) {
	type file struct {
		Name string `json:"name"`
	}
	tests := []struct {
		name     string
		want     []string
		wantErr  bool
		mockBody []file
		mockCode int
	}{
		{
			name:     "success - 0 targets",
			want:     nil,
			wantErr:  false,
			mockCode: http.StatusOK,
			mockBody: []file{},
		},
		{
			name: "success - 1 target",
			want: []string{
				"test",
			},
			wantErr:  false,
			mockCode: http.StatusOK,
			mockBody: []file{
				{
					Name: "test",
				},
			},
		},
		{
			name: "success - 2 targets",
			want: []string{
				"test",
				"test2",
			},
			wantErr:  false,
			mockCode: http.StatusOK,
			mockBody: []file{
				{
					Name: "test",
				},
				{
					Name: "test2",
				},
			},
		},
		{
			name:     "failure - API error",
			want:     nil,
			wantErr:  true,
			mockCode: http.StatusInternalServerError,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := httpmock.NewJsonResponder(tt.mockCode, tt.mockBody)
			if err != nil {
				t.Fatalf("error creating mock response: %v", err)
			}
			httpmock.RegisterResponder("GET", "http://test/api/v4/projects/1/repository/tree?ref=main", resp)

			g := &gitlab{
				baseUrl:   "http://test",
				projectID: 1,
				token:     "test",
				client:    http.DefaultClient,
			}
			got, err := g.fetchFileList(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FetchFiles() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// The filelist and url are the same, so we HTTP responders can
// be created without much hassle
func Test_gitlab_fetchFiles(t *testing.T) {
	tests := []struct {
		name     string
		want     []globalTarget
		fileList []string
		wantErr  bool
		mockCode int
	}{
		{
			name:     "success - 0 targets",
			want:     nil,
			wantErr:  false,
			mockCode: http.StatusOK,
		},
		{
			name: "success - 1 target",
			want: []globalTarget{
				{
					Url:      "test",
					LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			fileList: []string{
				"test",
			},
			wantErr:  false,
			mockCode: http.StatusOK,
		},
		{
			name: "success - 2 targets",
			want: []globalTarget{
				{
					Url:      "test",
					LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Url:      "test2",
					LastSeen: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			fileList: []string{
				"test",
				"test2",
			},
			wantErr:  false,
			mockCode: http.StatusOK,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &gitlab{
		baseUrl:   "http://test",
		projectID: 1,
		token:     "test",
		client:    http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup mock responses
			for i, target := range tt.want {
				resp, err := httpmock.NewJsonResponder(tt.mockCode, target)
				if err != nil {
					t.Fatalf("error creating mock response: %v", err)
				}
				httpmock.RegisterResponder("GET", fmt.Sprintf("http://test/api/v4/projects/1/repository/files/%s/raw?ref=main", tt.fileList[i]), resp)
			}

			got, err := g.fetchFiles(context.Background(), tt.fileList)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("FetchFiles() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_gitlab_fetchFiles_error_cases(t *testing.T) {
	type mockResponses struct {
		response globalTarget
		err      bool
	}

	tests := []struct {
		name          string
		mockResponses []mockResponses
		fileList      []string
	}{
		{
			name: "failure - direct API error",
			mockResponses: []mockResponses{
				{
					err: true,
				},
			},
			fileList: []string{
				"test",
			},
		},
		{
			name: "failure - API error after one successful request",
			mockResponses: []mockResponses{
				{
					response: globalTarget{
						Url:      "test",
						LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					err: false,
				},
				{
					response: globalTarget{},
					err:      true,
				},
			},
			fileList: []string{
				"test",
				"test2-will-fail",
			},
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &gitlab{
		baseUrl:   "http://test",
		projectID: 1,
		token:     "test",
		client:    http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, target := range tt.mockResponses {
				if target.err {
					errResp := httpmock.NewStringResponder(http.StatusInternalServerError, "")
					httpmock.RegisterResponder("GET", fmt.Sprintf("http://test/api/v4/projects/1/repository/files/%s/raw?ref=main", tt.fileList[i]), errResp)
					continue
				}
				resp, err := httpmock.NewJsonResponder(http.StatusOK, target)
				if err != nil {
					t.Fatalf("error creating mock response: %v", err)
				}
				httpmock.RegisterResponder("GET", fmt.Sprintf("http://test/api/v4/projects/1/repository/files/%s/raw?ref=main", tt.fileList[i]), resp)
			}

			_, err := g.fetchFiles(context.Background(), tt.fileList)
			if err == nil {
				t.Fatalf("Expected error but got none.")
			}
		})
	}
}

func Test_gitlabTargetManager_refreshTargets(t *testing.T) {
	now := time.Now()
	tooOld := now.Add(-time.Hour * 2)

	tests := []struct {
		name            string
		mockTargets     []globalTarget
		expectedHealthy []globalTarget
		wantErr         bool
	}{
		{
			name:            "success with 0 targets",
			mockTargets:     []globalTarget{},
			expectedHealthy: []globalTarget{},
		},
		{
			name: "success with 1 healthy target",
			mockTargets: []globalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
			},
			expectedHealthy: []globalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
			},
		},
		{
			name: "success with 1 unhealthy target",
			mockTargets: []globalTarget{
				{
					Url:      "test",
					LastSeen: tooOld,
				},
			},
		},
		{
			name: "success with 1 healthy and 1 unhealthy targets",
			mockTargets: []globalTarget{
				{
					Url:      "test",
					LastSeen: now,
				},
				{
					Url:      "test2",
					LastSeen: tooOld,
				},
			},
			expectedHealthy: []globalTarget{
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
				gitlab:             newMockGitlab(tt.mockTargets, tt.wantErr),
				name:               "test",
				unhealthyThreshold: time.Hour,
			}
			if err := gtm.refreshTargets(context.Background()); (err != nil) != tt.wantErr {
				t.Fatalf("refreshTargets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockGitlab struct {
	targets []globalTarget
	err     error
}

func (m mockGitlab) PutFile(ctx context.Context, file GitlabFile) error {
	panic("implement me")
}

func (m mockGitlab) PostFile(ctx context.Context, f GitlabFile) error {
	panic("implement me")
}

func (m mockGitlab) FetchFiles(ctx context.Context) ([]globalTarget, error) {
	return m.targets, m.err
}

func (m mockGitlab) FetchFileList(ctx context.Context) ([]string, error) {
	panic("implement me")
}

func newMockGitlab(targets []globalTarget, err bool) Gitlab {
	var e error
	if err {
		e = fmt.Errorf("error")
	}
	return &mockGitlab{
		targets: targets,
		err:     e,
	}
}
