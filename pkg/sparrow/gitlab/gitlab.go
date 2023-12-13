package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func New(url, token string, pid int) Gitlab {
	return &Client{
		baseUrl:   url,
		token:     token,
		projectID: pid,
		client:    &http.Client{},
	}
}

// FetchFiles fetches the files from the global targets repository from the configured gitlab repository
func (g *Client) FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error) {
	log := logger.FromContext(ctx).With("name", "FetchFiles")
	fl, err := g.fetchFileList(ctx)
	if err != nil {
		log.Error("Failed to fetch files", "error", err)
		return nil, err
	}

	result, err := g.fetchFiles(ctx, fl)
	if err != nil {
		log.Error("Failed to fetch files", "error", err)
		return nil, err
	}
	log.Info("Successfully fetched all target files", "files", len(result))
	return result, nil
}

// fetchFiles fetches the files from the global targets repository from the configured gitlab repository
func (g *Client) fetchFiles(ctx context.Context, fl []string) ([]checks.GlobalTarget, error) {
	var result []checks.GlobalTarget
	log := logger.FromContext(ctx).With("name", "fetchFiles")
	log.Debug("Fetching global files")
	for _, f := range fl {
		// URL encode the name
		n := url.PathEscape(f)
		req, err := http.NewRequestWithContext(ctx,
			http.MethodGet,
			fmt.Sprintf("%s/api/v4/projects/%d/repository/files/%s/raw?ref=main", g.baseUrl, g.projectID, n),
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
			log.Error("Failed to fetch file", "file", f, "error", err)
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			log.Error("Failed to fetch file", "status", res.Status)
			return nil, fmt.Errorf("request failed, status is %s", res.Status)
		}

		defer res.Body.Close()
		var gt checks.GlobalTarget
		err = json.NewDecoder(res.Body).Decode(&gt)
		if err != nil {
			log.Error("Failed to decode file after fetching", "file", f, "error", err)
			return nil, err
		}

		log.Debug("Successfully fetched file", "file", f)
		result = append(result, gt)
	}
	return result, nil
}

// fetchFileList fetches the files from the global targets repository from the configured gitlab repository
func (g *Client) fetchFileList(ctx context.Context) ([]string, error) {
	log := logger.FromContext(ctx).With("name", "fetchFileList")
	log.Debug("Fetching global files")
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
func (g *Client) PutFile(ctx context.Context, body File) error {
	log := logger.FromContext(ctx).With("name", "AddRegistration")
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

	resp, err := g.client.Do(req)
	if err != nil {
		log.Error("Failed to push registration file", "error", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("Failed to push registration file", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}

// PostFile commits the current instance to the configured gitlab repository
// as a global target for other sparrow instances to discover
func (g *Client) PostFile(ctx context.Context, body File) error {
	log := logger.FromContext(ctx).With("name", "AddRegistration")
	log.Debug("Registering sparrow instance to gitlab")

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

	resp, err := g.client.Do(req)
	if err != nil {
		log.Error("Failed to push registration file", "error", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Error("Failed to push registration file", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}

type File struct {
	Branch        string              `json:"branch"`
	AuthorEmail   string              `json:"author_email"`
	AuthorName    string              `json:"author_name"`
	Content       checks.GlobalTarget `json:"content"`
	CommitMessage string              `json:"commit_message"`
	fileName      string
}

// Bytes returns the bytes of the File
func (g File) Bytes() ([]byte, error) {
	b, err := json.Marshal(g)
	return b, err
}
