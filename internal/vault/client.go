// Package vaultutil provides a small wrapper around the official
// HashiCorp Vault Go SDK focused on the two operations bacli needs:
//  1. Read static connection metadata from a KV mount (host, port, db name).
//  2. Fetch short‑livedDynamic credentials from the database secrets engine.
//
// It purposefully keeps a very small API surface so that callers don’t need to
// worry about the underlying Vault types.
package vault

import (
	"fmt"
	"os"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
)

// Client is a thin wrapper that embeds the official Vault API client.
// All higher‑level helper methods hang off this type.
type Client struct {
	*vault.Client
}

// NewClient builds a Vault client from the usual environment variables or the
// explicit address/token you pass.  TLS settings can be overridden via the
// standard VAULT_* env vars (VAULT_CACERT, VAULT_CLIENT_CERT, …).
func NewClient(addr, token string) (*Client, error) {
	cfg := vault.DefaultConfig()

	// Resolve address (explicit beats env var beats SDK default).
	switch {
	case addr != "":
		cfg.Address = addr
	case os.Getenv("VAULT_ADDR") != "":
		cfg.Address = os.Getenv("VAULT_ADDR")
	}

	api, err := vault.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create API client: %w", err)
	}

	// Resolve token.
	switch {
	case token != "":
		api.SetToken(token)
	case os.Getenv("VAULT_TOKEN") != "":
		api.SetToken(os.Getenv("VAULT_TOKEN"))
	}

	return &Client{api}, nil
}

// -----------------------------------------------------------------------------
// Types returned to the caller
// -----------------------------------------------------------------------------

// ConnMeta holds static connection information that does NOT change between
// dumps (host, port, default database name).
// Your KV document should store these as strings so that mapping is simple.
//
//	path "config/pg-db1" --> {"host":"pg-db1","port":"5432","database":"postgres"}
//
// The struct tags guide mapstructure decoding.
type ConnMeta struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Database string `mapstructure:"database"`
}

// DynCreds holds the user/pass pair Vault generates along with the TTL that
// tells you when the credentials expire.
//
// These come from a `database/creds/<role>` read.
type DynCreds struct {
	Username string
	Password string
	TTL      time.Duration
}

// DBConnection aggregates everything a backup routine needs in a single value.
type DBConnection struct {
	ConnMeta
	DynCreds
}

// -----------------------------------------------------------------------------
// High‑level helper methods
// -----------------------------------------------------------------------------

// ReadConnMeta fetches a KV‑v2 secret and decodes the data block into ConnMeta.
// The function transparently handles KV‑v1 as well.
func (c *Client) ReadConnMeta(path string) (ConnMeta, error) {
	sec, err := c.Logical().Read(path)
	if err != nil {
		return ConnMeta{}, fmt.Errorf("vault read %q: %w", path, err)
	}
	if sec == nil || sec.Data == nil {
		return ConnMeta{}, fmt.Errorf("no data at %q", path)
	}

	// KV‑v2 nests real fields under "data".
	raw := sec.Data
	if inner, ok := sec.Data["data"].(map[string]any); ok {
		raw = inner
	}

	var cm ConnMeta
	if err := mapstructure.Decode(raw, &cm); err != nil {
		return ConnMeta{}, fmt.Errorf("decode KV secret %q: %w", path, err)
	}
	return cm, nil
}

// FetchDynCreds hits database/creds/<role> and returns the username/password
// and TTL in a structured value.
func (c *Client) FetchDynCreds(role string) (DynCreds, error) {
	sec, err := c.Logical().Read("database/creds/" + role)
	if err != nil {
		return DynCreds{}, fmt.Errorf("read creds %q: %w", role, err)
	}
	if sec == nil || sec.Data == nil {
		return DynCreds{}, fmt.Errorf("no data for role %q", role)
	}

	user, uOK := sec.Data["username"].(string)
	pass, pOK := sec.Data["password"].(string)
	if !uOK || !pOK {
		return DynCreds{}, fmt.Errorf("unexpected fields in creds for %q: %+v", role, sec.Data)
	}

	return DynCreds{
		Username: user,
		Password: pass,
		TTL:      time.Duration(sec.LeaseDuration) * time.Second,
	}, nil
}

// FullDBConnection is a convenience helper: given a KV path and a Role, it
// returns the combined static + dynamic information your backup code can use
// directly.
func (c *Client) FullDBConnection(kvPath, role string) (DBConnection, error) {
	meta, err := c.ReadConnMeta(kvPath)
	if err != nil {
		return DBConnection{}, err
	}
	creds, err := c.FetchDynCreds(role)
	if err != nil {
		return DBConnection{}, err
	}
	return DBConnection{ConnMeta: meta, DynCreds: creds}, nil
}
