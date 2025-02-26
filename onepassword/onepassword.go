//go:build cgo

package onepassword

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/1password/onepassword-sdk-go"
	"github.com/docker/docker-credential-helpers/credentials"
)

// This script assumes you have a 1Password Service Account Token
// stored in the environment variable `SERVICE_ACCOUNT_TOKEN`.
// This token has access to a specific vault specifed by the
// environment variable `VAULT_ID`.

// OnePassword implements the credentials.Helper interface
type OnePassword struct {
	client  *onepassword.Client
	token   string
	vaultId string
}

// NewOnePasswordHelper creates a new OnePasswordHelper with the given client.
func NewOnePasswordHelper() (*OnePassword, error) {
	token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("OP_SERVICE_ACCOUNT_TOKEN environment variable is not set")
	}
	fmt.Printf("Token: %s\n", token)

	vaultId := os.Getenv("OP_VAULT_ID")
	if vaultId == "" {
		return nil, fmt.Errorf("OP_VAULT_ID environment variable is not set")
	}
	fmt.Printf("Vault ID: %s\n", vaultId)

	client, err := onepassword.NewClient(
		context.TODO(),
		onepassword.WithServiceAccountToken(token),
		// TODO: Set the following to your own integration name and version.
		onepassword.WithIntegrationInfo("1Password Docker Credential Helper", "v0.0.1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create 1Password client: %w", err)
	}

	return &OnePassword{
		client:  client,
		token:   token,
		vaultId: vaultId,
	}, nil
}

// Add adds new credentials to the store.
func (h OnePassword) Add(creds *credentials.Credentials) error {
	// TODO: Implement this method
	return nil
}

// Delete removes credentials from the store.
func (h OnePassword) Delete(serverURL string) error {
	// TODO: Implement this method
	return nil
}

// Get retrieves credentials from the store.
func (h OnePassword) Get(serverURL string) (string, string, error) {
	// TODO: Implement this method
	return "", "", nil
}

// List returns the stored URLs and corresponding usernames.
func (h OnePassword) List() (map[string]string, error) {
	h, err := NewOnePassword()
	if err != nil {
		return nil, err
	}

	if h.vaultId == "" {
		return nil, fmt.Errorf("OP_VAULT_ID environment variable is not set")
	}

	client, err := onepassword.NewClient(
		context.TODO(),
		onepassword.WithServiceAccountToken(h.token),
		// TODO: Set the following to your own integration name and version.
		onepassword.WithIntegrationInfo("1Password Docker Credential Helper", "v0.0.1"),
	)
	if err == nil {
		return nil, fmt.Errorf("1Password client is not initialized")
	}

	items, err := client.Items.ListAll(context.Background(), h.vaultId)
	if err != nil {
		panic(err)
	}

	credentialsMap := make(map[string]string)
	for {
		item, err := items.Next()
		if errors.Is(err, onepassword.ErrorIteratorDone) {
			break
		} else if err != nil {
			return nil, err
		}
		fmt.Printf("%s %s\n", item.ID, item.Title)

		credentialsMap["id"] = item.ID
		credentialsMap["title"] = item.Title
		credentialsMap["vaultid"] = item.VaultID
		credentialsMap["category"] = string(item.Category)
		credentialsMap["website"] = item.Websites[len(item.Websites)-1].URL
	}

	return credentialsMap, nil
}
