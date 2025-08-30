package vault

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	vault "github.com/hashicorp/vault/api"
)

const (
	approleSecretIDPath = "auth/approle/role/%s/secret-id"
	approleLoginPath    = "auth/approle/login"
)

// ErrClientInit indicates failure to initialize the Vault API client.
var ErrClientInit = errors.New("vault client initialization failed")

type Option func(*config)

type config struct {
	address     string
	token       string
	approleName string
}

type Client struct {
	// The Vault Client
	api    *vault.Client
	config *config
}
type DynamicCredentials struct {
	Username string
	Password string
	TTL      time.Duration
}
type StaticCredentials struct {
	Host     string
	Port     int
	Database string
}

// Credentials
type Credentials struct {
	Static  StaticCredentials
	Dynamic DynamicCredentials
}

// -------------------------------------------------------------------------------
// TODO: 1. Initiate the client with a context.Context
// TODO: 2. Retreive secrets (host, port, database)
// TODO: 2.1. Default values for credentials retreived from environment variables
// TODO: 3. Retreive temporal username and password.
// -------------------------------------------------------------------------------

// TODO: This function initializes the Vault client
// 			 It uses defaults values, and uses functional options
// 			 to set the address and token.
// 			`WithToken` is a functional option that sets the token for authentication.

func WithAddress(address string) Option {
	return func(c *config) {
		c.address = address
	}
}

func WithToken(token string) Option {
	return func(c *config) {
		c.token = token
	}
}

func WithAppRole(approleName string) Option {
	return func(c *config) {
		c.approleName = approleName
	}
}

// NewClient creates and initializes a Vault Client using provided options.
// It will perform AppRole login if roleID and roleName are both set, otherwise
// a static token (from env or WithToken) is used.
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	// Build default config from environment
	cfg := &config{
		address: os.Getenv("VAULT_ADDR"),
		token:   os.Getenv("VAULT_TOKEN"),
	}
	// Apply user options
	for _, opt := range opts {
		opt(cfg)
	}

	// Prepare Vault API client config
	apiCfg := vault.DefaultConfig()
	if cfg.address != "" {
		apiCfg.Address = cfg.address
	}

	api, err := vault.NewClient(apiCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault API client: %w", err)
	}

	client := &Client{api: api, config: cfg}

	// Set initial token for static auth
	if cfg.token != "" {
		client.api.SetToken(cfg.token)
	}

	// Perform AppRole login if configured
	if cfg.approleName != "" {
		if err := client.loginAppRole(ctx); err != nil {
			return nil, fmt.Errorf("AppRole login failed: %w", err)
		}
	}

	return client, nil
}

// loginAppRole performs AppRole login using only the configured roleName.
// It fetches the role_id and generates a secret_id automatically.
func (c *Client) loginAppRole(ctx context.Context) error {
	// 1. Fetch RoleID
	roleIDPath := fmt.Sprintf("auth/approle/role/%s/role-id", c.config.approleName)
	roleIDSecret, err := c.api.Logical().ReadWithContext(ctx, roleIDPath)
	if err != nil {
		return fmt.Errorf("fetch role_id: %w", err)
	}
	if roleIDSecret == nil || roleIDSecret.Data["role_id"] == nil {
		return fmt.Errorf("no role_id returned from %s", roleIDPath)
	}
	roleID, ok := roleIDSecret.Data["role_id"].(string)
	if !ok || roleID == "" {
		return fmt.Errorf("invalid role_id format at %s", roleIDPath)
	}
	fmt.Println("-------------------------------------------------------------")
	fmt.Println(roleID)
	fmt.Println("-------------------------------------------------------------")

	// 2. Generate SecretID
	secretIDPath := fmt.Sprintf("auth/approle/role/%s/secret-id", c.config.approleName)
	secretIDResp, err := c.api.Logical().WriteWithContext(ctx, secretIDPath, nil)
	if err != nil {
		return fmt.Errorf("generate secret_id: %w", err)
	}
	if secretIDResp == nil || secretIDResp.Data["secret_id"] == nil {
		return fmt.Errorf("no secret_id returned from %s", secretIDPath)
	}
	secretID, ok := secretIDResp.Data["secret_id"].(string)
	if !ok || secretID == "" {
		return fmt.Errorf("invalid secret_id format at %s", secretIDPath)
	}

	// 3. Login with RoleID + SecretID
	loginData := map[string]any{
		"role_id":   roleID,
		"secret_id": secretID,
	}
	loginResp, err := c.api.Logical().WriteWithContext(ctx, approleLoginPath, loginData)
	if err != nil {
		return fmt.Errorf("approle login request: %w", err)
	}
	if loginResp.Auth == nil || loginResp.Auth.ClientToken == "" {
		return fmt.Errorf("no token in login response")
	}

	// Set the new token
	c.api.SetToken(loginResp.Auth.ClientToken)
	return nil
}

// TODO: This function retrieves the static credentials from the Vault
// 			 it uses the path to the secret as an argument
// NOTE: I need to minimize the usage of hardcoded values

// // Get the static credentials from the Vault
// func (client *Client) GetStaticCredentials(
// 	ctx context.Context,
// 	path string,
// ) (StaticCredentials, error) {
// 	// Read the static credentials from the Vault
// 	secret, err := client.api.Logical().Read(path)
// 	if err != nil {
// 		return StaticCredentials{}, err
// 	}
// 	if secret == nil {
// 		return StaticCredentials{}, fmt.Errorf("no data found at path: %s", path)
// 	}
//
// 	var staticCreds StaticCredentials
// 	err = mapstructure.Decode(secret.Data, &staticCreds)
// 	if err != nil {
// 		return StaticCredentials{}, err
// 	}
//
// 	return staticCreds, nil
// }

// TODO: This function retrieves the dynamic credentials from the Vault
// 			 it uses the role name as an argument
// NOTE: I need to minimize the usage of hardcoded values
// NOTE: I need to use a more generic way to get the role name

// Get the dynamic credentials from the Vault
// using the role name, [username, password]
func (client *Client) GetDynamicCredentials(
	ctx context.Context,
	role string,
) (DynamicCredentials, error) {
	// Read the dynamic credentials from the Vault
	secret, err := client.api.Logical().ReadWithContext(ctx, role)
	if err != nil {
		return DynamicCredentials{}, err
	}
	if secret == nil {
		return DynamicCredentials{}, fmt.Errorf("no data found at path: %s", role)
	}
	user, userOK := secret.Data["username"].(string)
	pass, passOK := secret.Data["password"].(string)
	if !userOK || !passOK {
		return DynamicCredentials{}, fmt.Errorf("invalid data format at path: %s", role)
	}
	var dynamicCreds DynamicCredentials
	dynamicCreds.Username = user
	dynamicCreds.Password = pass
	dynamicCreds.TTL = time.Duration(secret.LeaseDuration) * time.Second
	return dynamicCreds, nil
}

// GetCredentials retrieves both static and dynamic credentials from the Vault
// IDEA: I need something cleaner
// func GetCredentials(address, token string) (*Credentials, error) {
// 	client, err := Connect(address, token)
//
// 	// Get static credentials
// 	staticCreds, err := client.GetStaticCredentials(context.Background(), path)
// 	if err != nil {
// 		return Credentials{}, err
// 	}
//
// 	// Get dynamic credentials
// 	dynamicCreds, err := client.GetDynamicCredentials(context.Background(), role)
// 	if err != nil {
// 		return Credentials{}, err
// 	}
//
// 	return &Credentials{
// 		Static:  staticCreds,
// 		Dynamic: dynamicCreds,
// 	}, nil
// }
