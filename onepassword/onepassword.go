//go:build cgo

package onepassword

import (
    "context"
    "os"

    "github.com/docker/docker-credential-helpers/credentials"
    "github.com/1password/onepassword-sdk-go"
)

// This script assumes you have a 1Password Service Account Token
// stored in the environment variable `SERVICE_ACCOUNT_TOKEN`.
// This token has access to a specific vault specifed by the
// environment variable `VAULT_ID`.

// OnePassword implements the credentials.Helper interface
type OnePassword struct {
    client *onepassword.Client    
}

// NewOnePasswordHelper creates a new OnePasswordHelper with the given client.
func NewOnePasswordHelper(client *onepassword.Client) *OnePassword {
    return &OnePassword{client: client}
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
    items, err := h.client.GetItems(context.Background())
    if err != nil {
        return nil, err
    }

    credentialsMap := make(map[string]string)
    for _, item := range items {
        if item.Type != "login" {
            fmt.Printf("item type %s is not supported", item.type)
            // return nil, fmt.Errorf("item type %s is not supported", item.type)
        } else {
            credentialsMap[item.Username] = item.Username
            credentialsMap[item.Password] = item.Password
            credentialsMap[item.URL] = item.URL
        }

        
    }

    return credentialsMap, nil
}


