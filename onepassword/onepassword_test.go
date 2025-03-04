package onepassword

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/stretchr/testify/assert"
)

func setup() {
	token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	if token == "" {
		fmt.Errorf("OP_SERVICE_ACCOUNT_TOKEN environment variable is not set")
	}
	fmt.Printf("Token: %s\n", token)

	vaultId := os.Getenv("OP_VAULT_ID")
	if vaultId == "" {
		fmt.Errorf("OP_VAULT_ID environment variable is not set")
	}
	fmt.Printf("Vault ID: %s\n", vaultId)

	os.Setenv("OP_SERVICE_ACCOUNT_TOKEN", token)
	os.Setenv("OP_VAULT_ID", vaultId)
}

func teardown() {
	os.Unsetenv("OP_SERVICE_ACCOUNT_TOKEN")
	os.Unsetenv("OP_VAULT_ID")
}

func TestNewOnePasswordHelper(t *testing.T) {
	setup()

	defer teardown()

	helper, err := NewOnePasswordHelper()
	assert.NoError(t, err)
	assert.NotNil(t, helper)
}

func TestAdd(t *testing.T) {
	setup()
	defer teardown()

	helper, err := NewOnePasswordHelper()
	assert.NoError(t, err)

	creds := &credentials.Credentials{
		ServerURL: "https://TESTING.com",
		Username:  "test-user",
		Secret:    "test-password",
	}

	err = helper.Add(creds)
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	setup()
	defer teardown()

	helper, err := NewOnePasswordHelper()
	assert.NoError(t, err)

	err = helper.Delete("https://TESTING.com")
	assert.NoError(t, err)
}

func TestGet(t *testing.T) {
	setup()
	defer teardown()

	helper, err := NewOnePasswordHelper()
	assert.NoError(t, err)

	username, password, err := helper.Get("https://TESTING.com")
	assert.NoError(t, err)
	assert.Equal(t, "test-user", username)
	assert.Equal(t, "test-password", password)
}

func TestList(t *testing.T) {
	setup()
	defer teardown()

	helper, err := NewOnePasswordHelper()
	assert.NoError(t, err)

	credsMap, err := helper.List()
	assert.NoError(t, err)
	assert.NotEmpty(t, credsMap)
}
