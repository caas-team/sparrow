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

package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote"
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
			name:     "success - 0 targets with 1 file",
			want:     nil,
			wantErr:  false,
			mockCode: http.StatusOK,
			mockBody: []file{
				{
					Name: "README.md",
				},
			},
		},
		{
			name: "success - 1 target",
			want: []string{
				"test.json",
			},
			wantErr:  false,
			mockCode: http.StatusOK,
			mockBody: []file{
				{
					Name: "test.json",
				},
			},
		},
		{
			name: "success - 2 targets",
			want: []string{
				"test.json",
				"test2.json",
			},
			wantErr:  false,
			mockCode: http.StatusOK,
			mockBody: []file{
				{
					Name: "test.json",
				},
				{
					Name: "test2.json",
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
			httpmock.RegisterResponder("GET", fmt.Sprintf("http://test/api/v4/projects/1/repository/tree?order_by=id&pagination=keyset&per_page=%d&ref=main&sort=asc", paginationPerPage), resp)

			g := &client{
				config: Config{
					BaseURL:   "http://test",
					ProjectID: 1,
					Token:     "test",
					Branch:    fallbackBranch,
				},
				client: http.DefaultClient,
			}
			got, err := g.fetchFileList(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("FetchFiles() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// The filelist and url are the same, so we HTTP responders can
// be created without much hassle
func Test_gitlab_FetchFiles(t *testing.T) {
	type file struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name     string
		want     []checks.GlobalTarget
		fileList []file
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
			want: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			fileList: []file{
				{
					Name: "test.json",
				},
			},
			wantErr:  false,
			mockCode: http.StatusOK,
		},
		{
			name: "success - 2 targets",
			want: []checks.GlobalTarget{
				{
					Url:      "test",
					LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Url:      "test2",
					LastSeen: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			fileList: []file{
				{
					Name: "test.json",
				},
				{
					Name: "test2.json",
				},
			},
			wantErr:  false,
			mockCode: http.StatusOK,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &client{
		config: Config{
			BaseURL:   "http://test",
			ProjectID: 1,
			Token:     "test",
			Branch:    fallbackBranch,
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup mock responses
			for i, target := range tt.want {
				resp, err := httpmock.NewJsonResponder(tt.mockCode, target)
				if err != nil {
					t.Fatalf("error creating mock response: %v", err)
				}
				httpmock.RegisterResponder("GET", fmt.Sprintf("http://test/api/v4/projects/1/repository/files/%s/raw?ref=main", tt.fileList[i].Name), resp)
			}

			resp, err := httpmock.NewJsonResponder(tt.mockCode, tt.fileList)
			if err != nil {
				t.Fatalf("error creating mock response: %v", err)
			}
			httpmock.RegisterResponder("GET", fmt.Sprintf("http://test/api/v4/projects/1/repository/tree?order_by=id&pagination=keyset&per_page=%d&ref=main&sort=asc", paginationPerPage), resp)

			got, err := g.FetchFiles(context.Background())
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
	type file struct {
		Name string `json:"name"`
	}
	type mockResponses struct {
		response checks.GlobalTarget
		err      bool
	}

	tests := []struct {
		name          string
		mockResponses []mockResponses
		fileList      []file
	}{
		{
			name: "failure - direct API error",
			mockResponses: []mockResponses{
				{
					err: true,
				},
			},
			fileList: []file{
				{
					Name: "test",
				},
			},
		},
		{
			name: "failure - API error after one successful request",
			mockResponses: []mockResponses{
				{
					response: checks.GlobalTarget{
						Url:      "test",
						LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					err: false,
				},
				{
					response: checks.GlobalTarget{},
					err:      true,
				},
			},
			fileList: []file{
				{Name: "test"},
				{Name: "test2-will-fail"},
			},
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &client{
		config: Config{
			BaseURL:   "http://test",
			ProjectID: 1,
			Token:     "test",
			Branch:    fallbackBranch,
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, target := range tt.mockResponses {
				if target.err {
					errResp := httpmock.NewStringResponder(http.StatusInternalServerError, "")
					httpmock.RegisterResponder("GET", fmt.Sprintf("http://test/api/v4/projects/1/repository/files/%s/raw?ref=main", tt.fileList[i].Name), errResp)
					continue
				}
				resp, err := httpmock.NewJsonResponder(http.StatusOK, target)
				if err != nil {
					t.Fatalf("error creating mock response: %v", err)
				}
				httpmock.RegisterResponder("GET", fmt.Sprintf("http://test/api/v4/projects/1/repository/files/%s/raw?ref=main", tt.fileList[i].Name), resp)
			}

			_, err := g.FetchFiles(context.Background())
			if err == nil {
				t.Fatalf("Expected error but got none.")
			}
		})
	}
}

func TestClient_PutFile(t *testing.T) { //nolint:dupl // no need to refactor yet
	now := time.Now()
	tests := []struct {
		name     string
		file     remote.File
		mockCode int
		wantErr  bool
	}{
		{
			name: "success",
			file: remote.File{
				AuthorEmail: "test@sparrow",
				AuthorName:  "sparrpw",
				Content: checks.GlobalTarget{
					Url:      "https://test.de",
					LastSeen: now,
				},
				CommitMessage: "test-commit",
				Name:          "test.de.json",
			},
			mockCode: http.StatusOK,
		},
		{
			name: "failure - API error",
			file: remote.File{
				AuthorEmail: "test@sparrow",
				AuthorName:  "sparrpw",
				Content: checks.GlobalTarget{
					Url:      "https://test.de",
					LastSeen: now,
				},
				CommitMessage: "test-commit",
				Name:          "test.de.json",
			},
			mockCode: http.StatusInternalServerError,
			wantErr:  true,
		},
		{
			name:    "failure - empty file",
			wantErr: true,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &client{
		config: Config{
			BaseURL:   "http://test",
			ProjectID: 1,
			Token:     "test",
			Branch:    fallbackBranch,
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				resp := httpmock.NewStringResponder(tt.mockCode, "")
				httpmock.RegisterResponder("PUT", fmt.Sprintf("http://test/api/v4/projects/1/repository/files/%s", tt.file.Name), resp)
			} else {
				resp, err := httpmock.NewJsonResponder(tt.mockCode, tt.file)
				if err != nil {
					t.Fatalf("error creating mock response: %v", err)
				}
				httpmock.RegisterResponder("PUT", fmt.Sprintf("http://test/api/v4/projects/1/repository/files/%s", tt.file.Name), resp)
			}

			if err := g.PutFile(context.Background(), tt.file); (err != nil) != tt.wantErr {
				t.Fatalf("PutFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_PostFile(t *testing.T) { //nolint:dupl // no need to refactor yet
	now := time.Now()
	tests := []struct {
		name     string
		file     remote.File
		mockCode int
		wantErr  bool
	}{
		{
			name: "success",
			file: remote.File{
				AuthorEmail: "test@sparrow",
				AuthorName:  "sparrpw",
				Content: checks.GlobalTarget{
					Url:      "https://test.de",
					LastSeen: now,
				},
				CommitMessage: "test-commit",
				Name:          "test.de.json",
			},
			mockCode: http.StatusCreated,
		},
		{
			name: "failure - API error",
			file: remote.File{
				AuthorEmail: "test@sparrow",
				AuthorName:  "sparrpw",
				Content: checks.GlobalTarget{
					Url:      "https://test.de",
					LastSeen: now,
				},
				CommitMessage: "test-commit",
				Name:          "test.de.json",
			},
			mockCode: http.StatusInternalServerError,
			wantErr:  true,
		},
		{
			name:    "failure - empty file",
			wantErr: true,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &client{
		config: Config{
			BaseURL:   "http://test",
			ProjectID: 1,
			Token:     "test",
			Branch:    fallbackBranch,
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				resp := httpmock.NewStringResponder(tt.mockCode, "")
				httpmock.RegisterResponder("POST", fmt.Sprintf("http://test/api/v4/projects/1/repository/files/%s", tt.file.Name), resp)
			} else {
				resp, err := httpmock.NewJsonResponder(tt.mockCode, tt.file)
				if err != nil {
					t.Fatalf("error creating mock response: %v", err)
				}
				httpmock.RegisterResponder("POST", fmt.Sprintf("http://test/api/v4/projects/1/repository/files/%s", tt.file.Name), resp)
			}

			if err := g.PostFile(context.Background(), tt.file); (err != nil) != tt.wantErr {
				t.Fatalf("PostFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_DeleteFile(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		mockCode int
		wantErr  bool
	}{
		{
			name:     "success",
			fileName: "test.de.json",
			mockCode: http.StatusNoContent,
		},
		{
			name:     "failure - API error",
			fileName: "test.de.json",
			mockCode: http.StatusInternalServerError,
			wantErr:  true,
		},
		{
			name:     "failure - empty file",
			wantErr:  true,
			fileName: "",
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	projID := 1
	g := &client{
		config: Config{
			BaseURL:   "http://test",
			ProjectID: 1,
			Token:     "test",
			Branch:    fallbackBranch,
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httpmock.NewStringResponder(tt.mockCode, "")
			httpmock.RegisterResponder("DELETE", fmt.Sprintf("http://test/api/v4/projects/%d/repository/files/%s", projID, tt.fileName), resp)

			f := remote.File{
				Name:          tt.fileName,
				CommitMessage: "Deleted registration file",
				AuthorName:    "sparrow-test",
				AuthorEmail:   "sparrow-test@sparrow",
			}
			if err := g.DeleteFile(context.Background(), f); (err != nil) != tt.wantErr {
				t.Fatalf("DeleteFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_fetchDefaultBranch(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		want     string
		response any
	}{
		{
			name: "success",
			code: http.StatusOK,
			want: "master",
			response: []branch{
				{
					Name:    "master",
					Default: true,
				},
			},
		},
		{
			name: "success - multiple branches",
			code: http.StatusOK,
			want: "release",
			response: []branch{
				{
					Name:    "master",
					Default: false,
				},
				{
					Name:    "release",
					Default: true,
				},
			},
		},
		{
			name: "success - multiple branches without default",
			code: http.StatusOK,
			want: fallbackBranch,
			response: []branch{
				{
					Name:    "master",
					Default: false,
				},
				{
					Name:    "release",
					Default: false,
				},
			},
		},
		{
			name: "failure - API error",
			code: http.StatusInternalServerError,
			want: fallbackBranch,
		},
		{
			name: "failure - invalid response",
			code: http.StatusOK,
			want: fallbackBranch,
			response: struct {
				Invalid bool `json:"invalid"`
			}{
				Invalid: true,
			},
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &client{
		config: Config{
			BaseURL:   "http://test",
			ProjectID: 1,
			Token:     "test",
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := httpmock.NewJsonResponder(tt.code, tt.response)
			if err != nil {
				t.Fatalf("error creating mock response: %v", err)
			}
			httpmock.RegisterResponder(http.MethodGet, "http://test/api/v4/projects/1/repository/branches", resp)

			got := g.fetchDefaultBranch()
			if got != tt.want {
				t.Errorf("(*client).fetchDefaultBranch() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_fetchNextFileList(t *testing.T) {
	type mockRespFile struct {
		Name string `json:"name"`
	}
	type mockResponder struct {
		reqUrl     string
		linkHeader string
		statusCode int
		response   []mockRespFile
	}

	tests := []struct {
		name    string
		mock    []mockResponder
		want    []string
		wantErr bool
	}{
		{
			name: "success - no pagination",
			mock: []mockResponder{
				{
					reqUrl:     "https://test.de/pagination",
					linkHeader: "",
					statusCode: http.StatusOK,
					response:   []mockRespFile{{Name: "file1.json"}, {Name: "file2.json"}},
				},
			},
			want: []string{
				"file1.json",
				"file2.json",
			},
			wantErr: false,
		},
		{
			name: "success - with pagination",
			mock: []mockResponder{
				{
					reqUrl:     "https://test.de/pagination",
					linkHeader: "<https://test.de/pagination?page=2>; rel=\"next\"",
					statusCode: http.StatusOK,
					response:   []mockRespFile{{Name: "file1.json"}},
				},
				{
					reqUrl:     "https://test.de/pagination?page=2",
					linkHeader: "<https://test.de/pagination?page=3>; rel=\"next\"",
					statusCode: http.StatusOK,
					response:   []mockRespFile{{Name: "file2.json"}},
				},
				{
					reqUrl:     "https://test.de/pagination?page=3",
					linkHeader: "",
					statusCode: http.StatusOK,
					response:   []mockRespFile{{Name: "file3.json"}},
				},
			},
			want: []string{
				"file1.json",
				"file2.json",
				"file3.json",
			},
			wantErr: false,
		},
		{
			name: "fail - status code nok while paginated requests",
			mock: []mockResponder{
				{
					reqUrl:     "https://test.de/pagination",
					linkHeader: "<https://test.de/pagination?page=2>; rel=\"next\"",
					statusCode: http.StatusOK,
					response:   []mockRespFile{{Name: "file1.json"}},
				},
				{
					reqUrl:     "https://test.de/pagination?page=2",
					linkHeader: "",
					statusCode: http.StatusBadRequest,
					response:   []mockRespFile{{Name: "file2.json"}},
				},
			},
			want:    []string{},
			wantErr: true,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	c := &client{
		config: Config{
			BaseURL:   "https://test.de",
			ProjectID: 1,
			Token:     "test",
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// prepare http mock responder for paginated requests
			for _, responder := range tt.mock {
				httpmock.RegisterResponder(http.MethodGet, responder.reqUrl, func(req *http.Request) (*http.Response, error) {
					// Check if header are properly set
					token := req.Header.Get("PRIVATE-TOKEN")
					cType := req.Header.Get("Content-Type")
					if token == "" || cType == "" {
						t.Error("Some header not properly set", "PRIVATE-TOKEN", token != "", "Content-Type", cType != "")
					}

					resp, err := httpmock.NewJsonResponse(responder.statusCode, responder.response)

					// Add link header for next page (pagination)
					resp.Header.Set(linkHeader, responder.linkHeader)
					return resp, err
				})
			}

			got, err := c.fetchNextFileList(context.Background(), tt.mock[0].reqUrl)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("fetchNextFileList() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatalf("fetchNextFileList() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("fetchNextFileList() got = %v, want %v", got, tt.want)
			}
		})
	}
}
