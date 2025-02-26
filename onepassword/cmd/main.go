//go:build cgo

package main

import (
	"log"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/onepassword"
)

func main() {
	helper, err := onepassword.NewOnePasswordHelper()
	if err != nil {
		log.Fatal(err)
	}
	credentials.Serve(helper)
}
