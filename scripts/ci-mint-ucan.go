//go:build ignore

// Mint a UCAN for CI multinode tests. Usage:
//
//	go run scripts/ci-mint-ucan.go <identity.key> <audience-did> <resource> <actions>
//
// Example:
//
//	go run scripts/ci-mint-ucan.go /tmp/node1/identity.key did:key:z... repo:ci-test read,write
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lakshmanpatel/gitant/internal/identity"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintf(os.Stderr, "usage: %s <identity.key> <audience-did> <resource> <actions>\n", os.Args[0])
		os.Exit(2)
	}

	id, err := identity.LoadIdentity(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "load identity: %v\n", err)
		os.Exit(1)
	}

	audience := os.Args[2]
	resource := os.Args[3]
	actions := strings.Split(os.Args[4], ",")

	caps := []identity.Capability{{Resource: resource, Actions: actions}}
	ucan := identity.NewUCAN(id.DID, audience, caps, 1*time.Hour)
	token, err := ucan.Sign(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign ucan: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(token)
}
