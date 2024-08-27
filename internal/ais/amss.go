package ais

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-cleanhttp"
)

type AMSSClient struct {
	client *http.Client
	url    *url.URL
	auth   string
}

func NewAMSSClient(config AMSSConfig) (*AMSSClient, error) {
	u, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("NewAMSSClient: parse URL: %w", err)
	}
	return &AMSSClient{
		client: cleanhttp.DefaultPooledClient(),
		url:    u,
		auth:   fmt.Sprintf("ApiKey %s:%s", config.User, config.Key),
	}, nil
}

func (c *AMSSClient) GetAIPPath(ctx context.Context, aipUUID string) (string, error) {
	u := c.url.JoinPath("api/v2/file", aipUUID)
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("GetAIPPath: new request: %w", err)
	}
	req.Header.Set("Authorization", c.auth)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GetAIPPath: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GetAIPPath: unexpected status code: %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("GetAIPPath: decode response: %w", err)
	}

	value, ok := data["current_path"]
	if !ok {
		return "", fmt.Errorf("GetAIPPath: current_path not found in response")
	}

	path, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("GetAIPPath: current_path is not a string")
	}

	return path, nil
}

func (c *AMSSClient) DownloadAIPFile(ctx context.Context, aipUUID, path string, writer io.Writer) error {
	u := c.url.JoinPath("api/v2/file", aipUUID, "extract_file")
	query := url.Values{}
	query.Set("relative_path_to_file", path)
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return fmt.Errorf("DownloadAIPFile: new request: %w", err)
	}
	req.Header.Set("Authorization", c.auth)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("DownloadAIPFile: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DownloadAIPFile: unexpected status code: %d", resp.StatusCode)
	}

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("DownloadAIPFile: copy file: %w", err)
	}

	return nil
}
