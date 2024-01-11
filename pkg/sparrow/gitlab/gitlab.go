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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
)

// Gitlab handles interaction with a gitlab repository containing
// the global targets for the Sparrow instance
type Gitlab interface {
	// FetchFiles fetches the files from the global targets repository
	FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error)
	// PutFile updates the file in the repository
	PutFile(ctx context.Context, file File) error
	// PostFile creates the file in the repository
	PostFile(ctx context.Context, file File) error
	// DeleteFile deletes the file from the repository
	DeleteFile(ctx context.Context, file File) error
}

// Client implements Gitlab
type Client struct {
	// the base URL of the gitlab instance
	baseUrl string
	// the ID of the project containing the global targets
	projectID int
	// the token used to authenticate with the gitlab instance
	token  string
	client *http.Client
}

// DeleteFile deletes the file matching the filename from the configured
// gitlab repository
func (g *Client) DeleteFile(ctx context.Context, file File) error { //nolint:gocritic // no performance concerns yet
	log := logger.FromContext(ctx).With("file", file)

	if file.fileName == "" {
		return fmt.Errorf("filename is empty")
	}

	log.Debug("Deleting file from gitlab")
	n := url.PathEscape(file.fileName)
	b, err := file.Bytes()
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return err
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodDelete,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s", g.baseUrl, g.projectID, n),
		bytes.NewBuffer(b),
	)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return err
	}

	req.Header.Add("PRIVATE-TOKEN", g.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := g.client.Do(req) //nolint:bodyclose // closed in defer
	if err != nil {
		log.Error("Failed to delete file", "error", err)
		return err
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusNoContent {
		log.Error("Failed to delete file", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}

// File represents a File manipulation operation via the Gitlab API
type File struct {
	Branch        string              `json:"branch"`
	AuthorEmail   string              `json:"author_email"`
	AuthorName    string              `json:"author_name"`
	Content       checks.GlobalTarget `json:"content"`
	CommitMessage string              `json:"commit_message"`
	fileName      string
}

func New(baseURL, token string, pid int) Gitlab {
	return &Client{
		baseUrl:   baseURL,
		token:     token,
		projectID: pid,
		client:    &http.Client{},
	}
}

// FetchFiles fetches the files from the global targets repository from the configured gitlab repository
func (g *Client) FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error) {
	log := logger.FromContext(ctx)
	fl, err := g.fetchFileList(ctx)
	if err != nil {
		log.Error("Failed to fetch files", "error", err)
		return nil, err
	}
	// TODO: pop all non json files from fl
	var result []checks.GlobalTarget
	for _, f := range fl {
		gl, err := g.fetchFile(ctx, f)
		if err != nil {
			log.Error("Failed fetching files", "error", err)
			return nil, err
		}
		result = append(result, gl)
	}
	log.Info("Successfully fetched all target files", "files", len(result))
	return result, nil
}

// fetchFile fetches the file from the global targets repository from the configured gitlab repository
func (g *Client) fetchFile(ctx context.Context, f string) (checks.GlobalTarget, error) {
	log := logger.FromContext(ctx).With("file", f)
	var res checks.GlobalTarget
	// URL encode the name
	n := url.PathEscape(f)
	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s/raw?ref=main", g.baseUrl, g.projectID, n),
		http.NoBody,
	)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return res, err
	}
	req.Header.Add("PRIVATE-TOKEN", g.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := g.client.Do(req) //nolint:bodyclose // closed in defer
	if err != nil {
		log.Error("Failed to fetch file", "error", err)
		return res, err
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Error("Failed to fetch file", "status", resp.Status)
		return res, fmt.Errorf("request failed, status is %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		log.Error("Failed to decode file after fetching", "error", err)
		return res, err
	}

	log.Debug("Successfully fetched file")
	return res, nil
}

// fetchFileList fetches the filenames from the global targets repository from the configured gitlab repository,
// so they may be fetched individually
func (g *Client) fetchFileList(ctx context.Context) ([]string, error) {
	log := logger.FromContext(ctx)
	log.Debug("Fetching file list from gitlab")
	type file struct {
		Name string `json:"name"`
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/tree?ref=main", g.baseUrl, g.projectID),
		http.NoBody,
	)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return nil, err
	}

	req.Header.Add("PRIVATE-TOKEN", g.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := g.client.Do(req) //nolint:bodyclose // closed in defer
	if err != nil {
		log.Error("Failed to fetch file list", "error", err)
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Error("Failed to fetch file list", "status", resp.Status)
		return nil, fmt.Errorf("request failed, status is %s", resp.Status)
	}

	var fl []file
	err = json.NewDecoder(resp.Body).Decode(&fl)
	if err != nil {
		log.Error("Failed to decode file list", "error", err)
		return nil, err
	}

	var result []string
	for _, f := range fl {
		if strings.HasSuffix(f.Name, ".json") {
			result = append(result, f.Name)
		}
	}

	log.Debug("Successfully fetched file list", "files", len(result))
	return result, nil
}

// PutFile commits the current instance to the configured gitlab repository
// as a global target for other sparrow instances to discover
func (g *Client) PutFile(ctx context.Context, body File) error { //nolint: dupl,gocritic // no need to refactor yet
	log := logger.FromContext(ctx)
	log.Debug("Registering sparrow instance to gitlab")

	// chose method based on whether the registration has already happened
	n := url.PathEscape(body.fileName)
	b, err := body.Bytes()
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return err
	}
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPut,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s", g.baseUrl, g.projectID, n),
		bytes.NewBuffer(b),
	)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return err
	}

	req.Header.Add("PRIVATE-TOKEN", g.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := g.client.Do(req) //nolint:bodyclose // closed in defer
	if err != nil {
		log.Error("Failed to push registration file", "error", err)
		return err
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Error("Failed to push registration file", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}

// PostFile commits the current instance to the configured gitlab repository
// as a global target for other sparrow instances to discover
func (g *Client) PostFile(ctx context.Context, body File) error { //nolint:dupl,gocritic // no need to refactor yet
	log := logger.FromContext(ctx)
	log.Debug("Posting registration file to gitlab")

	// chose method based on whether the registration has already happened
	n := url.PathEscape(body.fileName)
	b, err := body.Bytes()
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return err
	}
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s", g.baseUrl, g.projectID, n),
		bytes.NewBuffer(b),
	)
	if err != nil {
		log.Error("Failed to create request", "error", err)
		return err
	}

	req.Header.Add("PRIVATE-TOKEN", g.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := g.client.Do(req) //nolint:bodyclose // closed in defer
	if err != nil {
		log.Error("Failed to post file", "error", err)
		return err
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		log.Error("Failed to post file", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}

// Bytes returns the File as a byte array. The Content
// is base64 encoded for Gitlab API compatibility.
func (g *File) Bytes() ([]byte, error) {
	content, err := json.Marshal(g.Content)
	if err != nil {
		return nil, err
	}

	// base64 encode the content
	enc := base64.NewEncoder(base64.StdEncoding, bytes.NewBuffer(content))
	_, err = enc.Write(content)
	_ = enc.Close()

	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{
		"branch":         g.Branch,
		"author_email":   g.AuthorEmail,
		"author_name":    g.AuthorName,
		"content":        string(content),
		"commit_message": g.CommitMessage,
	})
}

// SetFileName sets the filename of the File
func (g *File) SetFileName(name string) {
	g.fileName = name
}
