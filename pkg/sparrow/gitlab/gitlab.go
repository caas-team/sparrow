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

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
)

// Gitlab handles interaction with a gitlab repository containing
// the global targets for the Sparrow instance
type Gitlab interface {
	FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error)
	PutFile(ctx context.Context, file File) error
	PostFile(ctx context.Context, file File) error
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
	log := logger.FromContext(ctx)
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
		log.Error("Failed to fetch file", "file", f, "error", err)
		return res, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Error("Failed to fetch file", "status", resp.Status)
		return res, fmt.Errorf("request failed, status is %s", resp.Status)
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		log.Error("Failed to decode file after fetching", "file", f, "error", err)
		return res, err
	}

	log.Debug("Successfully fetched file", "file", f)
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

	res, err := g.client.Do(req)
	if err != nil {
		log.Error("Failed to fetch file list", "error", err)
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		log.Error("Failed to fetch file list", "status", res.Status)
		return nil, fmt.Errorf("request failed, status is %s", res.Status)
	}

	defer res.Body.Close()
	var fl []file
	err = json.NewDecoder(res.Body).Decode(&fl)
	if err != nil {
		log.Error("Failed to decode file list", "error", err)
		return nil, err
	}

	var result []string
	for _, f := range fl {
		result = append(result, f.Name)
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

// File represents a File manipulation operation via the Gitlab API
type File struct {
	Branch        string              `json:"branch"`
	AuthorEmail   string              `json:"author_email"`
	AuthorName    string              `json:"author_name"`
	Content       checks.GlobalTarget `json:"content"`
	CommitMessage string              `json:"commit_message"`
	fileName      string
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
