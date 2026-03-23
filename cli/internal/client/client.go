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
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
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

func (c *Client) ListApps() ([]map[string]any, error) {
	resp, err := c.do("GET", "/api/apps", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out []map[string]any
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

func (c *Client) CreateApp(name string) (map[string]any, error) {
	resp, err := c.do("POST", "/api/apps", map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out map[string]any
	return out, json.NewDecoder(resp.Body).Decode(&out)
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
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out []map[string]any
	return out, json.NewDecoder(resp.Body).Decode(&out)
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
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out map[string]any
	return out, json.NewDecoder(resp.Body).Decode(&out)
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
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out []map[string]any
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

func (c *Client) SetVar(envID int64, key, value string) (map[string]any, error) {
	resp, err := c.do("POST", fmt.Sprintf("/api/envs/%d/vars", envID), map[string]string{"key": key, "value": value})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out map[string]any
	return out, json.NewDecoder(resp.Body).Decode(&out)
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
