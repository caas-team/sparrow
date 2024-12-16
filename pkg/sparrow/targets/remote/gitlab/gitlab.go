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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote"

	"github.com/caas-team/sparrow/internal/logger"
)

// The amount of items the paginated request to gitlab should return
const paginationPerPage = 30

var _ remote.Interactor = (*client)(nil)

// client is the implementation of the remote.Interactor for gitlab
type client struct {
	// config contains the configuration for the gitlab client
	config Config
	// client is the http client used to interact with the gitlab instance
	client *http.Client
}

// Config contains the configuration for the gitlab client
type Config struct {
	// BaseURL is the URL of the gitlab instance
	BaseURL string `yaml:"baseUrl" mapstructure:"baseUrl"`
	// Token is the personal access token used to authenticate with the gitlab instance
	Token string `yaml:"token" mapstructure:"token"`
	// ProjectID is the ID of the project in the gitlab instance that contains the global targets
	ProjectID int `yaml:"projectId" mapstructure:"projectId"`
	// Branch is the branch to use for the gitlab repository
	Branch string `yaml:"branch" mapstructure:"branch"`
}

// New creates a new gitlab client
func New(cfg Config) remote.Interactor {
	c := &client{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	if c.config.Branch == "" {
		c.config.Branch = c.fetchDefaultBranch()
	}
	return c
}

// FetchFiles fetches the files from the global targets repository from the configured gitlab repository
func (c *client) FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error) {
	log := logger.FromContext(ctx)
	fl, err := c.fetchFileList(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Failed to fetch files", "error", err)
		return nil, err
	}

	var result []checks.GlobalTarget
	for _, f := range fl {
		gl, err := c.fetchFile(ctx, f)
		if err != nil {
			log.ErrorContext(ctx, "Failed fetching files", "error", err)
			return nil, err
		}
		result = append(result, gl)
	}
	log.InfoContext(ctx, "Successfully fetched all target files", "files", len(result))
	return result, nil
}

// fetchFile fetches the file from the global targets repository from the configured gitlab repository
func (c *client) fetchFile(ctx context.Context, f string) (checks.GlobalTarget, error) {
	log := logger.FromContext(ctx).With("file", f)
	var res checks.GlobalTarget
	// URL encode the name
	n := url.PathEscape(f)
	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s/raw", c.config.BaseURL, c.config.ProjectID, n),
		http.NoBody,
	)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create request", "error", err)
		return res, err
	}
	req.Header.Add("PRIVATE-TOKEN", c.config.Token)
	req.Header.Add("Content-Type", "application/json")
	query := req.URL.Query()
	query.Add("ref", c.config.Branch)
	req.URL.RawQuery = query.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		log.ErrorContext(ctx, "Failed to fetch file", "error", err)
		return res, err
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		log.ErrorContext(ctx, "Failed to fetch file", "status", resp.Status)
		return res, fmt.Errorf("request failed, status is %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		log.ErrorContext(ctx, "Failed to decode file after fetching", "error", err)
		return res, err
	}

	log.DebugContext(ctx, "Successfully fetched file")
	return res, nil
}

// fetchFileList fetches the filenames from the global targets repository from the configured gitlab repository,
// so they may be fetched individually
func (c *client) fetchFileList(ctx context.Context) ([]string, error) {
	log := logger.FromContext(ctx)
	log.DebugContext(ctx, "Preparing to fetch file list from gitlab")

	rawUrl := fmt.Sprintf("%s/api/v4/projects/%d/repository/tree", c.config.BaseURL, c.config.ProjectID)
	reqUrl, err := url.Parse(rawUrl)
	if err != nil {
		log.ErrorContext(ctx, "Could not parse GitLab API repository URL", "url", rawUrl, "error", err)
	}
	query := reqUrl.Query()
	query.Set("pagination", "keyset")
	query.Set("per_page", strconv.Itoa(paginationPerPage))
	query.Set("order_by", "id")
	query.Set("sort", "asc")

	query.Set("ref", c.config.Branch)
	reqUrl.RawQuery = query.Encode()

	return c.fetchNextFileList(ctx, reqUrl.String())
}

// fetchNextFileList is fetching files from GitLab.
// Gitlab pagination is handled recursively.
func (c *client) fetchNextFileList(ctx context.Context, reqUrl string) ([]string, error) {
	log := logger.FromContext(ctx)
	log.DebugContext(ctx, "Fetching file list page from gitlab")

	type file struct {
		Name string `json:"name"`
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, http.NoBody)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create request", "error", err)
		return nil, err
	}
	req.Header.Add("PRIVATE-TOKEN", c.config.Token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		log.ErrorContext(ctx, "Failed to fetch file list", "error", err)
		return nil, err
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		log.ErrorContext(ctx, "Failed to fetch file list", "status", resp.Status)
		return nil, fmt.Errorf("request failed, status is %s", resp.Status)
	}

	var fl []file
	err = json.NewDecoder(resp.Body).Decode(&fl)
	if err != nil {
		log.ErrorContext(ctx, "Failed to decode file list", "error", err)
		return nil, err
	}

	var files []string
	for _, f := range fl {
		if strings.HasSuffix(f.Name, ".json") {
			files = append(files, f.Name)
		}
	}

	if nextLink := getNextLink(resp.Header); nextLink != "" {
		nextFiles, err := c.fetchNextFileList(ctx, nextLink)
		if err != nil {
			return nil, err
		}
		log.DebugContext(ctx, "Successfully fetched next file page, adding to file list")
		files = append(files, nextFiles...)
	}

	log.DebugContext(ctx, "Successfully fetched file list recursively", "files", len(files))
	return files, nil
}

// PutFile commits the current instance to the configured gitlab repository
// as a global target for other sparrow instances to discover
func (c *client) PutFile(ctx context.Context, file remote.File) error { //nolint: dupl,gocritic // no need to refactor yet
	log := logger.FromContext(ctx)
	log.DebugContext(ctx, "Registering sparrow instance to gitlab")

	// chose method based on whether the registration has already happened
	n := url.PathEscape(file.Name)
	b, err := file.Serialize(c.config.Branch)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create request", "error", err)
		return err
	}
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPut,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s", c.config.BaseURL, c.config.ProjectID, n),
		bytes.NewBuffer(b),
	)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create request", "error", err)
		return err
	}

	req.Header.Add("PRIVATE-TOKEN", c.config.Token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		log.ErrorContext(ctx, "Failed to push registration file", "error", err)
		return err
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	// Unfortunately the Gitlab API does not give any information about what went wrong and just returns a 400 Bad Request in case of
	// an error while committing the file. A probable cause for this is that the branch does not exist or is protected.
	// The documentation documents some more possible errors, but not all of them: https://docs.gitlab.com/ee/api/repository_files.html#update-existing-file-in-repository
	// Therefore we just check for the status code and return an error if it is not 200 OK.
	// This is not ideal, but the best we can do with the current API without implementing a full blown error handling mechanism.
	if resp.StatusCode != http.StatusOK {
		log.ErrorContext(ctx, "Failed to push registration file", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}

// PostFile commits the current instance to the configured gitlab repository
// as a global target for other sparrow instances to discover
func (c *client) PostFile(ctx context.Context, file remote.File) error { //nolint:dupl,gocritic // no need to refactor yet
	log := logger.FromContext(ctx)
	log.DebugContext(ctx, "Posting registration file to gitlab")

	// chose method based on whether the registration has already happened
	n := url.PathEscape(file.Name)
	b, err := file.Serialize(c.config.Branch)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create request", "error", err)
		return err
	}
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s", c.config.BaseURL, c.config.ProjectID, n),
		bytes.NewBuffer(b),
	)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create request", "error", err)
		return err
	}

	req.Header.Add("PRIVATE-TOKEN", c.config.Token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		log.ErrorContext(ctx, "Failed to post file", "error", err)
		return err
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	// Unfortunately the Gitlab API does not give any information about what went wrong and just returns a 400 Bad Request in case of
	// an error while committing the file. A probable cause for this is that the branch does not exist or is protected.
	// The documentation documents some more possible errors, but not all of them: https://docs.gitlab.com/ee/api/repository_files.html#update-existing-file-in-repository
	// Therefore we just check for the status code and return an error if it is not 200 OK.
	// This is not ideal, but the best we can do with the current API without implementing a full blown error handling mechanism.
	if resp.StatusCode != http.StatusCreated {
		log.ErrorContext(ctx, "Failed to post file", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}

// DeleteFile deletes the file matching the filename from the configured gitlab repository
func (c *client) DeleteFile(ctx context.Context, file remote.File) error { //nolint:gocritic // no performance concerns yet
	log := logger.FromContext(ctx).With("file", file)

	if file.Name == "" {
		return fmt.Errorf("filename is empty")
	}

	log.DebugContext(ctx, "Deleting file from gitlab")
	n := url.PathEscape(file.Name)
	b, err := file.Serialize(c.config.Branch)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create request", "error", err)
		return err
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodDelete,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s", c.config.BaseURL, c.config.ProjectID, n),
		bytes.NewBuffer(b),
	)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create request", "error", err)
		return err
	}

	req.Header.Add("PRIVATE-TOKEN", c.config.Token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		log.ErrorContext(ctx, "Failed to delete file", "error", err)
		return err
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusNoContent {
		log.ErrorContext(ctx, "Failed to delete file", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}

// fallbackBranch is the branch to use if no default branch is found
const fallbackBranch = "main"

// branch is a struct to decode the branches from the gitlab API
type branch struct {
	// Name is the name of the branch
	Name string `json:"name"`
	// Default is true if the branch is the default branch
	Default bool `json:"default"`
}

// fetchDefaultBranch fetches the default branch from the gitlab API
func (c *client) fetchDefaultBranch() string {
	ctx := context.Background()
	log := logger.FromContext(ctx).With("gitlab", c.config.BaseURL)

	log.DebugContext(ctx, "No branch configured, fetching default branch")
	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v4/projects/%d/repository/branches", c.config.BaseURL, c.config.ProjectID),
		http.NoBody,
	)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create request", "error", err)
		return fallbackBranch
	}

	req.Header.Add("PRIVATE-TOKEN", c.config.Token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		log.ErrorContext(ctx, "Failed to fetch branches", "error", err)
		return fallbackBranch
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		log.ErrorContext(ctx, "Failed to fetch branches", "status", resp.Status)
		return fallbackBranch
	}

	var branches []branch
	err = json.NewDecoder(resp.Body).Decode(&branches)
	if err != nil {
		log.ErrorContext(ctx, "Failed to decode branches", "error", err)
		return fallbackBranch
	}

	for _, b := range branches {
		if b.Default {
			log.DebugContext(ctx, "Successfully fetched default branch", "branch", b.Name)
			return b.Name
		}
	}

	log.WarnContext(ctx, "No default branch found, using fallback", "fallback", fallbackBranch)
	return fallbackBranch
}
