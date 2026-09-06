package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

type Client struct {
	Base   *url.URL
	APIKey string
	HTTP   *http.Client
}

func New(base, key string) (*Client, error) {
	if base == "" {
		return nil, fmt.Errorf("api base URL is required (run `envious login --api-base=...`)")
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid api base URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid api base URL %q: scheme must be http or https", base)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("invalid api base URL %q: missing host", base)
	}
	if key == "" {
		return nil, fmt.Errorf("api key is required (run `envious login --api-key=...`)")
	}
	return &Client{
		Base:   u,
		APIKey: key,
		HTTP:   &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (c *Client) do(method, p string, body any) (*http.Response, error) {
	u := *c.Base
	u.Path = path.Join(u.Path, p)
	u.RawQuery = ""
	return c.doURL(method, &u, body)
}

func (c *Client) doWithQuery(method, p string, query url.Values, body any) (*http.Response, error) {
	u := *c.Base
	u.Path = path.Join(u.Path, p)
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return c.doURL(method, &u, body)
}

func (c *Client) doURL(method string, u *url.URL, body any) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, u.String(), r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.HTTP.Do(req)
}

func (c *Client) Login(base, key string) {
	c.APIKey = key
}

// errorForStatus reads a bounded error body so CLI errors preserve the
// server's {"error": "..."} message instead of a bare status code.
func errorForStatus(resp *http.Response) error {
	const maxErrBody = 4 * 1024
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))
	var serverErr struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &serverErr); err == nil && serverErr.Error != "" {
		return fmt.Errorf("request failed: status %d: %s", resp.StatusCode, serverErr.Error)
	}
	if msg := string(bytes.TrimSpace(body)); msg != "" {
		return fmt.Errorf("request failed: status %d: %s", resp.StatusCode, msg)
	}
	return fmt.Errorf("request failed: status %d", resp.StatusCode)
}

func decodeJSON(resp *http.Response, out any) error {
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) ListApps() ([]map[string]any, error) {
	resp, err := c.do("GET", "/api/apps", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errorForStatus(resp)
	}
	var out []map[string]any
	if err := decodeJSON(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateApp(name string) (map[string]any, error) {
	resp, err := c.do("POST", "/api/apps", map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return nil, errorForStatus(resp)
	}
	var out map[string]any
	if err := decodeJSON(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) DeleteApp(id int64) error {
	resp, err := c.do("DELETE", fmt.Sprintf("/api/apps/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) ListEnvs(appID int64) ([]map[string]any, error) {
	var q url.Values
	if appID != 0 {
		q = url.Values{}
		q.Set("app_id", fmt.Sprintf("%d", appID))
	}
	resp, err := c.doWithQuery("GET", "/api/envs", q, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errorForStatus(resp)
	}
	var out []map[string]any
	if err := decodeJSON(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateEnv(appID int64, name string) (map[string]any, error) {
	body := map[string]any{"name": name}
	if appID != 0 {
		body["app_id"] = appID
	}
	resp, err := c.do("POST", "/api/envs", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return nil, errorForStatus(resp)
	}
	var out map[string]any
	if err := decodeJSON(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) DeleteEnv(id int64) error {
	resp, err := c.do("DELETE", fmt.Sprintf("/api/envs/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) ListVars(envID int64) ([]map[string]any, error) {
	resp, err := c.do("GET", fmt.Sprintf("/api/envs/%d/vars", envID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errorForStatus(resp)
	}
	var out []map[string]any
	if err := decodeJSON(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) SetVar(envID int64, key, value string) (map[string]any, error) {
	resp, err := c.do("POST", fmt.Sprintf("/api/envs/%d/vars", envID), map[string]string{"key": key, "value": value})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, errorForStatus(resp)
	}
	var out map[string]any
	if err := decodeJSON(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) DeleteVarByID(id int64) error {
	resp, err := c.do("DELETE", fmt.Sprintf("/api/vars/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
