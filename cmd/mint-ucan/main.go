package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lakshmanpatel/gitant/internal/identity"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <server-did> <resource> [actions...]\n", os.Args[0])
		os.Exit(2)
	}
	serverDID := os.Args[1]
	resource := os.Args[2]
	actions := []string{"write"}
	if len(os.Args) > 3 {
		actions = os.Args[3:]
	}

	id, err := identity.NewIdentity()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate identity: %v\n", err)
		os.Exit(1)
	}

	caps := []identity.Capability{{Resource: resource, Actions: actions}}
	ucan := identity.NewUCAN(id.DID, serverDID, caps, 1*time.Hour)
	token, err := ucan.Sign(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign ucan: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(token)
	fmt.Fprintf(os.Stderr, "issuer=%s\n", id.DID)
	fmt.Fprintf(os.Stderr, "nonce=%s\n", ucan.Nonce)
}
