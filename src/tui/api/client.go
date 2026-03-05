package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type Config struct {
	ServerURL string `json:"serverUrl"`
	Token     string `json:"token"`
	Email     string `json:"email"`
}

type Client struct {
	Config     Config
	HTTPClient *http.Client
	configPath string
}

func NewClient() *Client {
	c := &Client{
		HTTPClient: &http.Client{},
	}
	c.loadConfig()
	return c
}

func (c *Client) configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".imitsu")
}

func (c *Client) loadConfig() {
	c.configPath = filepath.Join(c.configDir(), "config.json")
	c.Config = Config{ServerURL: "http://localhost:3100"}

	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &c.Config)
}

func (c *Client) SaveConfig() error {
	dir := c.configDir()
	os.MkdirAll(dir, 0700)
	data, err := json.MarshalIndent(c.Config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.configPath, data, 0600)
}

func (c *Client) IsLoggedIn() bool {
	return c.Config.Token != ""
}

func (c *Client) request(method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.Config.ServerURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Config.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(respBody, &errResp)
		if errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("request failed: %d", resp.StatusCode)
	}

	return respBody, nil
}

// Auth

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type LoginResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

func (c *Client) Login(email, password string) (*LoginResponse, error) {
	data, err := c.request("POST", "/api/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return nil, err
	}

	var resp LoginResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	c.Config.Token = resp.Token
	c.Config.Email = email
	c.SaveConfig()

	return &resp, nil
}

func (c *Client) Register(email, name, password string) (*User, error) {
	data, err := c.request("POST", "/api/auth/register", map[string]string{
		"email":    email,
		"name":     name,
		"password": password,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		User User `json:"user"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp.User, nil
}

func (c *Client) WhoAmI() (*User, error) {
	data, err := c.request("GET", "/api/auth/me", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		User User `json:"user"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp.User, nil
}

func (c *Client) Logout() {
	c.Config.Token = ""
	c.Config.Email = ""
	c.SaveConfig()
}

// Secrets

type Secret struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Value     string `json:"value,omitempty"`
	Category  string `json:"category"`
	CreatedBy string `json:"created_by"`
	Version   int    `json:"version"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (c *Client) ListSecrets() ([]Secret, error) {
	data, err := c.request("GET", "/api/secrets", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Secrets []Secret `json:"secrets"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Secrets, nil
}

func (c *Client) GetSecret(id string) (*Secret, error) {
	data, err := c.request("GET", "/api/secrets/"+id, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Secret Secret `json:"secret"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp.Secret, nil
}

func (c *Client) CreateSecret(name, value, category string) (*Secret, error) {
	data, err := c.request("POST", "/api/secrets", map[string]string{
		"name":     name,
		"value":    value,
		"category": category,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Secret Secret `json:"secret"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp.Secret, nil
}

func (c *Client) UpdateSecret(id, value string) error {
	_, err := c.request("PUT", "/api/secrets/"+id, map[string]string{
		"value": value,
	})
	return err
}

func (c *Client) DeleteSecret(id string) error {
	_, err := c.request("DELETE", "/api/secrets/"+id, nil)
	return err
}

type ExportedSecret struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Category string `json:"category"`
}

func (c *Client) ExportSecrets() ([]ExportedSecret, error) {
	data, err := c.request("GET", "/api/secrets/export", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Secrets []ExportedSecret `json:"secrets"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Secrets, nil
}

// Teams

type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MemberCount int    `json:"member_count"`
	CreatedAt   string `json:"created_at"`
}

type TeamMember struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

type TeamDetail struct {
	Team    Team         `json:"team"`
	Members []TeamMember `json:"members"`
}

func (c *Client) GetTeamDetails(id string) (*TeamDetail, error) {
	data, err := c.request("GET", "/api/teams/"+id, nil)
	if err != nil {
		return nil, err
	}

	var resp TeamDetail
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListTeams() ([]Team, error) {
	data, err := c.request("GET", "/api/teams", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Teams []Team `json:"teams"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Teams, nil
}

func (c *Client) ListUsers() ([]User, error) {
	data, err := c.request("GET", "/api/auth/users", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Users []User `json:"users"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Users, nil
}
