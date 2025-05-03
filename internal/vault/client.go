package vault

import (
	"fmt"
	"os"

	vault "github.com/hashicorp/vault/api"
)

// Client wraps the HashiCorp Vault API client.
type Client struct {
	*vault.Client
}
type Secret struct {
	Username string
	Password string
	Database string
	Host     string
	Port     string
}

// NewClient initializes a Vault client using the provided address and token.
// If addr or token are empty, VAULT_ADDR and VAULT_TOKEN environment variables are used.
func NewClient(addr, token string) (*Client, error) {
	config := vault.DefaultConfig()

	// Set Vault address
	if addr != "" {
		config.Address = addr
	} else if envAddr := os.Getenv("VAULT_ADDR"); envAddr != "" {
		config.Address = envAddr
	}

	// Create underlying API client
	apiClient, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault API client: %w", err)
	}

	// Set token
	if token != "" {
		apiClient.SetToken(token)
	} else if envToken := os.Getenv("VAULT_TOKEN"); envToken != "" {
		apiClient.SetToken(envToken)
	}

	return &Client{apiClient}, nil
}

// ReadSecret fetches a KV secret at the given path and unmarshals it into a Secret.
// It supports KV-v2 mounts (data under "data") and falls back to KV-v1 style.
func (c *Client) ReadSecret(path string) (*Secret, error) {
	res, err := c.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("error reading secret %q: %w", path, err)
	}
	if res == nil {
		return nil, fmt.Errorf("no secret found at %q", path)
	}

	var raw map[string]any
	// KV-v2 stores data under "data"
	if d, ok := res.Data["data"].(map[string]any); ok {
		raw = d
	} else {
		raw = res.Data
	}

	// Extract fields with type assertions
	username, ok := raw["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username field missing or not string in %q", path)
	}
	password, ok := raw["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password field missing or not string in %q", path)
	}
	host, ok := raw["host"].(string)
	if !ok {
		return nil, fmt.Errorf("host field missing or not string in %q", path)
	}
	port, ok := raw["port"].(string)
	if !ok {
		return nil, fmt.Errorf("port field missing or not string in %q", path)
	}
	database, ok := raw["database"].(string)
	if !ok {
		// database may be optional; default to empty
		database = ""
	}

	return &Secret{
		Username: username,
		Password: password,
		Host:     host,
		Port:     port,
		Database: database,
	}, nil
}
