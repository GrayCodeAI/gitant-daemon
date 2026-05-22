package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/lakshmanpatel/gitant/internal/cli"
	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage webhooks",
}

var webhookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered webhooks",
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result struct {
			Webhooks []struct {
				ID     string   `json:"id"`
				URL    string   `json:"url"`
				Events []string `json:"events"`
			} `json:"webhooks"`
			Total int `json:"total"`
		}
		if err := client.Get("/api/v1/webhooks", &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		for _, wh := range result.Webhooks {
			fmt.Printf("%s\t%s\t[%s]\n", wh.ID, wh.URL, strings.Join(wh.Events, ","))
		}
		fmt.Fprintf(os.Stderr, "%d webhook(s)\n", result.Total)
	},
}

var webhookRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new webhook",
	Run: func(cmd *cobra.Command, args []string) {
		url, _ := cmd.Flags().GetString("url")
		events, _ := cmd.Flags().GetString("events")
		secret, _ := cmd.Flags().GetString("secret")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		if url == "" || events == "" {
			fmt.Fprintln(os.Stderr, "Error: --url and --events are required")
			os.Exit(1)
		}

		client := cli.NewClient(daemonURL)
		req := map[string]interface{}{
			"url":    url,
			"events": strings.Split(events, ","),
		}
		if secret != "" {
			req["secret"] = secret
		}

		var result map[string]interface{}
		if err := client.Post("/api/v1/webhooks", req, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Registered webhook: %s\n", result["id"])
	},
}

var webhookDeleteCmd = &cobra.Command{
	Use:   "delete [webhook-id]",
	Short: "Delete a webhook",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		if err := client.Delete(fmt.Sprintf("/api/v1/webhooks/%s", args[0])); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted webhook %s\n", args[0])
	},
}

func init() {
	webhookListCmd.Flags().String("daemon-url", "", "Daemon URL (default: http://localhost:7777)")
	webhookRegisterCmd.Flags().String("url", "", "Webhook URL (required)")
	webhookRegisterCmd.Flags().String("events", "", "Comma-separated event types (required)")
	webhookRegisterCmd.Flags().String("secret", "", "Webhook secret")
	webhookRegisterCmd.Flags().String("daemon-url", "", "Daemon URL (default: http://localhost:7777)")
	webhookDeleteCmd.Flags().String("daemon-url", "", "Daemon URL (default: http://localhost:7777)")

	webhookCmd.AddCommand(webhookListCmd, webhookRegisterCmd, webhookDeleteCmd)
	rootCmd.AddCommand(webhookCmd)
}
