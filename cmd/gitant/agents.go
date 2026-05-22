package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/lakshmanpatel/gitant/internal/cli"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents and DID identities",
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known agents",
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result struct {
			Agents []struct {
				DID        string  `json:"did"`
				TrustScore float64 `json:"trust_score"`
				RepoCount  int     `json:"repos"`
				CommitCount int    `json:"commits"`
			} `json:"agents"`
			Total int `json:"total"`
		}
		if err := client.Get("/api/v1/agents", &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		for _, a := range result.Agents {
			fmt.Printf("%s\ttrust=%.2f\trepos=%d\tcommits=%d\n", a.DID, a.TrustScore, a.RepoCount, a.CommitCount)
		}
		fmt.Fprintf(os.Stderr, "%d agent(s)\n", result.Total)
	},
}

var agentShowCmd = &cobra.Command{
	Use:   "show [did]",
	Short: "Show agent details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Get(fmt.Sprintf("/api/v1/agents/%s", args[0]), &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		for k, v := range result {
			fmt.Printf("%s: %v\n", k, v)
		}
	},
}

var agentGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new DID identity",
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Post("/api/v1/agents/generate-did", nil, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated DID: %s\n", result["did"])
	},
}

var agentDelegateCmd = &cobra.Command{
	Use:   "delegate",
	Short: "Delegate capabilities to another agent",
	Run: func(cmd *cobra.Command, args []string) {
		did, _ := cmd.Flags().GetString("did")
		resource, _ := cmd.Flags().GetString("resource")
		actions, _ := cmd.Flags().GetString("actions")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		if did == "" || resource == "" || actions == "" {
			fmt.Fprintln(os.Stderr, "Error: --did, --resource, and --actions are required")
			os.Exit(1)
		}

		client := cli.NewClient(daemonURL)
		req := map[string]interface{}{
			"audience": did,
			"resource": resource,
			"actions":  strings.Split(actions, ","),
		}

		var result map[string]interface{}
		if err := client.Post("/api/v1/agents/"+did+"/delegate", req, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Delegated capabilities to %s\n", did)
		fmt.Printf("Token: %s\n", result["token"])
	},
}

var agentVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify a UCAN token",
	Run: func(cmd *cobra.Command, args []string) {
		token, _ := cmd.Flags().GetString("token")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		if token == "" {
			fmt.Fprintln(os.Stderr, "Error: --token is required")
			os.Exit(1)
		}

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Post("/api/v1/agents/verify", map[string]string{"token": token}, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Token valid: %v\n", result)
	},
}

func init() {
	for _, c := range []*cobra.Command{agentListCmd, agentShowCmd, agentGenerateCmd, agentDelegateCmd, agentVerifyCmd} {
		c.Flags().String("daemon-url", "", "Daemon URL (default: http://localhost:7777)")
	}
	agentDelegateCmd.Flags().String("did", "", "Target DID (required)")
	agentDelegateCmd.Flags().String("resource", "", "Resource (required)")
	agentDelegateCmd.Flags().String("actions", "", "Comma-separated actions (required)")
	agentVerifyCmd.Flags().String("token", "", "UCAN token to verify (required)")

	agentCmd.AddCommand(agentListCmd, agentShowCmd, agentGenerateCmd, agentDelegateCmd, agentVerifyCmd)
	rootCmd.AddCommand(agentCmd)
}
