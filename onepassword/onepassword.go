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
//
// Usage: echo https://docker.com | ./bin/build/docker-credential-onepassword get
// Usage: echo echo '{"ServerURL":"https://example.com","Username":"myuser","Secret":"mypassword"}' |  ./bin/build/docker-credential-onepassword store
// Usage: echo https://example.com | ./bin/build/docker-credential-onepassword erase
// Usage: ./bin/build/docker-credential-onepassword list

// OnePassword implements the credentials.Helper interface
type OnePassword struct {
	client  *onepassword.Client
	token   string
	vaultID string
	itemID  string
}

// NewOnePasswordHelper creates a new OnePasswordHelper with the given client.
func NewOnePasswordHelper() (*OnePassword, error) {
	fmt.Printf("Creating new helper\n")
	token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("OP_SERVICE_ACCOUNT_TOKEN environment variable is not set")
	}
	fmt.Printf("- Using token from Environment\n")

	vaultId := os.Getenv("OP_VAULT_ID")
	if vaultId == "" {
		return nil, fmt.Errorf("OP_VAULT_ID environment variable is not set")
	}
	fmt.Printf("- Using Vault ID from Environment\n")

	client, err := onepassword.NewClient(
		context.TODO(),
		onepassword.WithServiceAccountToken(token),
		// TODO: Set the following to your own integration name and version.
		onepassword.WithIntegrationInfo("1Password Docker Credential Helper", "v0.0.1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create 1Password client: %w", err)
	}

	fmt.Printf("- 1Password Client Created\n")

	return &OnePassword{
		client:  client,
		token:   token,
		vaultID: vaultId,
	}, nil
}

// Add adds new credentials to the store.
func (h *OnePassword) Add(creds *credentials.Credentials) error {
	fmt.Printf("Add: Adding credentials for Creds: %s\n", creds.ServerURL)

	if h == nil {
		newHelper, err := NewOnePasswordHelper()
		if err != nil {
			return err
		}
		h = newHelper
	}

	sectionID := "extraDetails"
	itemParams := onepassword.ItemCreateParams{
		Title:    "Login created with the 1Password SDK",
		Category: onepassword.ItemCategoryLogin,
		VaultID:  h.vaultID,
		Fields: []onepassword.ItemField{
			{
				ID:        "username",
				Title:     "username",
				Value:     creds.Username,
				FieldType: onepassword.ItemFieldTypeText,
			},
			{
				ID:        "password",
				Title:     "password",
				Value:     creds.Secret,
				FieldType: onepassword.ItemFieldTypeConcealed,
			},
		},
		Sections: []onepassword.ItemSection{
			{
				ID:    sectionID,
				Title: "This item stores Credentials for a Docker Archive",
			},
		},
		Tags: []string{"Docker Credential Helper", "Docker"},
		Websites: []onepassword.Website{
			{
				URL:              creds.ServerURL,
				AutofillBehavior: onepassword.AutofillBehaviorAnywhereOnWebsite,
				Label:            "Docker Archive",
			},
		},
	}

	// Creates a new item based on the structure definition above
	createdItem, err := h.client.Items().Create(context.Background(), itemParams)
	if err != nil {
		return err
	}

	// Retrieves the newly created item
	login, err := h.client.Items().Get(context.Background(), createdItem.VaultID, createdItem.ID)
	if err != nil {
		return err
	}

	fmt.Printf("- Item created: %s\n", login.ID)

	return nil
}

// Delete removes credentials from the store.
func (h *OnePassword) Delete(serverURL string) error {
	fmt.Printf("Deleting credentials for Website: %s\n", serverURL)

	if h == nil {
		newHelper, err := NewOnePasswordHelper()
		if err != nil {
			return err
		}
		h = newHelper
	}

	shouldReturn, err := hydrateItemIDByURL(h, serverURL)
	if !shouldReturn {
		return err
	}

	err = h.client.Items().Delete(context.Background(), h.vaultID, h.itemID)
	if err != nil {
		return err
	}

	return nil
}

// Get retrieves credentials from the store.
func (h *OnePassword) Get(serverURL string) (string, string, error) {
	fmt.Printf("Getting credentials for Website: %s\n", serverURL)
	if h == nil {
		newHelper, err := NewOnePasswordHelper()
		if err != nil {
			return "", "", err
		}
		h = newHelper
	}

	// Get the item ID if it is not already set
	// This is a backward reference by website (which is what Docker is looking for)
	// src - https://pkg.go.dev/github.com/docker/docker-credential-helpers/client
	fmt.Printf("- Getting item ID by Website\n")

	if h.itemID == "" {
		shouldReturn, err := hydrateItemIDByURL(h, serverURL)
		if !shouldReturn {
			return "", "", err
		}

		// ok now we have the item ID
		// we can get the username and password

		item, err := h.client.Items().Get(context.Background(), h.vaultID, h.itemID)
		if err != nil {
			panic(err)
		}

		if item.Category == onepassword.ItemCategoryLogin {
			fmt.Printf("- Item Title: %s\n", item.Title)
			var username, password string
			for _, field := range item.Fields {
				if field.Title == "username" {
					username = field.Value
				} else if field.Title == "password" {
					password = field.Value
				}
			}
			if username == "" || password == "" {
				return "", "", fmt.Errorf("username or password field is missing")
			}
			return username, password, nil
		} else {
			return "", "", fmt.Errorf("item is not a login item")
		}
	}

	return "", "", nil
}

func hydrateItemIDByURL(h *OnePassword, serverURL string) (bool, error) {
	fmt.Printf("Hydrating item ID by URL: %s\n", serverURL)

	items, err := h.client.Items().ListAll(context.Background(), h.vaultID)
	if err != nil {
		panic(err)
	}

	for {
		item, err := items.Next()

		if errors.Is(err, onepassword.ErrorIteratorDone) {
			break
		} else if err != nil {
			return false, err
		}

		// detect if item is active // TODO: Fix this later once we have a better way to detect active items
		fullItem, err := h.client.Items().Get(context.Background(), h.vaultID, item.ID)
		if err != nil {
			// Item not active. Skip
			fmt.Printf("- Item not active. Skipping: %s\n", fullItem.ID)
			break
		}

		for _, website := range item.Websites {
			if website.URL == serverURL {
				fmt.Printf("- Found item ID: %s - %s\n", item.ID, item.Websites[0].URL)
				h.itemID = item.ID
				return true, nil
			}
		}
	}

	return false, fmt.Errorf("Credentials not found for Website: %s", serverURL)
}

// List returns the stored URLs and corresponding usernames.
func (h *OnePassword) List() (map[string]string, error) {
	if h == nil {
		newHelper, err := NewOnePasswordHelper()
		if err != nil {
			return nil, err
		}
		h = newHelper
	}

	items, err := h.client.Items().ListAll(context.Background(), h.vaultID)
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
		fmt.Printf("-- %s %s\n", item.ID, item.Title)

		credentialsMap["id"] = item.ID
		credentialsMap["title"] = item.Title
		credentialsMap["vaultid"] = item.VaultID
		credentialsMap["category"] = string(item.Category)

		// Not sure if this is the right way to do this
		// Docker does not support multiple websites, but 1Password does.
		// anyway, i just grab the first website
		for _, website := range item.Websites {
			fmt.Printf("- Website: %s\n", website.URL)
			credentialsMap["website"] = website.URL
			break
		}

	}

	return credentialsMap, nil
}
