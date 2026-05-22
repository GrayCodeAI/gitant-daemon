package main

import (
	"fmt"
	"os"

	"github.com/lakshmanpatel/gitant/internal/cli"
	"github.com/spf13/cobra"
)

var ucanCmd = &cobra.Command{
	Use:   "ucan",
	Short: "Manage UCAN tokens",
}

var ucanRevokeCmd = &cobra.Command{
	Use:   "revoke [nonce]",
	Short: "Revoke a UCAN by nonce",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Post("/api/v1/ucan/revoke", map[string]string{"nonce": args[0]}, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Revoked UCAN with nonce: %s\n", args[0])
	},
}

var ucanListRevocationsCmd = &cobra.Command{
	Use:   "list-revocations",
	Short: "List all revoked UCAN nonces",
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result struct {
			Revocations []struct {
				Nonce     string `json:"nonce"`
				RevokedAt int64  `json:"revoked_at"`
			} `json:"revocations"`
			Total int `json:"total"`
		}
		if err := client.Get("/api/v1/ucan/revocations", &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if result.Total == 0 {
			fmt.Println("No revoked UCANs")
			return
		}

		for _, entry := range result.Revocations {
			fmt.Printf("%s\trevoked_at=%d\n", entry.Nonce, entry.RevokedAt)
		}
		fmt.Fprintf(os.Stderr, "%d revocation(s)\n", result.Total)
	},
}

func init() {
	for _, c := range []*cobra.Command{ucanRevokeCmd, ucanListRevocationsCmd} {
		c.Flags().String("daemon-url", "", "Daemon URL (default: http://localhost:7777)")
	}

	ucanCmd.AddCommand(ucanRevokeCmd, ucanListRevocationsCmd)
	rootCmd.AddCommand(ucanCmd)
}
