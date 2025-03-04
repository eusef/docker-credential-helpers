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

// OnePassword implements the credentials.Helper interface
type OnePassword struct {
	client  *onepassword.Client
	token   string
	vaultID string
	itemID  string
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

	fmt.Printf("1Password Client Created\n")

	return &OnePassword{
		client:  client,
		token:   token,
		vaultID: vaultId,
	}, nil
}

// Add adds new credentials to the store.
func (h *OnePassword) Add(creds *credentials.Credentials) error {
	println("Adding credentials for Creds: %s", creds.ServerURL)

	if h == nil {
		println("Creating new helper")
		newHelper, err := NewOnePasswordHelper()
		if err != nil {
			return err
		}
		h = newHelper
	}

	// [developer-docs.sdk.go.create-item]-start
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
	createdItem, err := h.client.Items.Create(context.Background(), itemParams)
	if err != nil {
		return err
	}

	// Retrieves the newly created item
	login, err := h.client.Items.Get(context.Background(), createdItem.VaultID, createdItem.ID)
	if err != nil {
		return err
	}

	// Retrieve TOTP code from an item
	for _, f := range login.Fields {
		if f.FieldType == onepassword.ItemFieldTypeTOTP {
			OTPFieldDetails := f.Details.OTP()
			if OTPFieldDetails.ErrorMessage == nil {
				fmt.Println(*OTPFieldDetails.Code)
			} else {
				panic(*OTPFieldDetails.ErrorMessage)
			}
		}
	}

	return nil
}

// Delete removes credentials from the store.
func (h *OnePassword) Delete(serverURL string) error {
	println("Deleting credentials for Website: %s", serverURL)

	if h == nil {
		println("Creating new helper")
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

	err = h.client.Items.Delete(context.Background(), h.vaultID, h.itemID)
	if err != nil {
		return err
	}

	return nil
}

// Get retrieves credentials from the store.
func (h *OnePassword) Get(serverURL string) (string, string, error) {
	println("Getting credentials for Website: %s", serverURL)
	if h == nil {
		println("Creating new helper")
		newHelper, err := NewOnePasswordHelper()
		if err != nil {
			return "", "", err
		}
		h = newHelper
	}

	// Get the item ID if it is not already set
	// This is a backward reference by website (which is what Docker is looking for)
	// src - https://pkg.go.dev/github.com/docker/docker-credential-helpers/client
	println("Getting item ID by Website")

	if h.itemID == "" {
		shouldReturn, err := hydrateItemIDByURL(h, serverURL)
		if !shouldReturn {
			return "", "", err
		}

		// ok now we have the item ID
		// we can get the username and password

		item, err := h.client.Items.Get(context.Background(), h.vaultID, h.itemID)
		if err != nil {
			panic(err)
		}

		if item.Category == onepassword.ItemCategoryLogin {
			println("Item: Title: %s", item.Title)
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
	items, err := h.client.Items.ListAll(context.Background(), h.vaultID)
	if err != nil {
		panic(err)
	}

	for {
		item, err := items.Next()
		if errors.Is(err, onepassword.ErrorIteratorDone) {
			break
		} else if err != nil {
			return true, err
		}

		for _, website := range item.Websites {
			if website.URL == serverURL {
				fmt.Printf("Found item ID: %s\n", item.ID)
				fmt.Printf("%s %s\n", item.ID, item.Title)
				h.itemID = item.ID
				break
			}
		}
	}
	return false, nil
}

// List returns the stored URLs and corresponding usernames.
func (h *OnePassword) List() (map[string]string, error) {
	if h == nil {
		println("Creating new helper")
		newHelper, err := NewOnePasswordHelper()
		if err != nil {
			return nil, err
		}
		h = newHelper
	}

	items, err := h.client.Items.ListAll(context.Background(), h.vaultID)
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

		for index, website := range item.Websites {
			fmt.Printf("Website: %s\n", website.URL)
			credentialsMap["website"+fmt.Sprint(index)] = website.URL
		}

	}

	return credentialsMap, nil
}
