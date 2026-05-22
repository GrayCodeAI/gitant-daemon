package main

import (
	"fmt"
	"os"

	"github.com/lakshmanpatel/gitant/internal/cli"
	"github.com/spf13/cobra"
)

var protectionCmd = &cobra.Command{
	Use:   "protection",
	Short: "Manage branch protection rules",
}

var protectionShowCmd = &cobra.Command{
	Use:   "show [repo] [branch]",
	Short: "Show protection rules for a branch",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		branch := args[1]
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Get(fmt.Sprintf("/api/v1/repos/%s/protections/%s", repo, branch), &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if protected, ok := result["protected"].(bool); ok && !protected {
			fmt.Printf("Branch %s is not protected\n", branch)
			return
		}

		fmt.Printf("Branch: %s\n", result["branch"])
		fmt.Printf("  Require PR:       %v\n", result["require_pr"])
		fmt.Printf("  Require Approval: %v\n", result["require_approval"])
		fmt.Printf("  No Force Push:    %v\n", result["no_force_push"])
	},
}

var protectionSetCmd = &cobra.Command{
	Use:   "set [repo] [branch]",
	Short: "Set protection rules for a branch",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		branch := args[1]
		daemonURL, _ := cmd.Flags().GetString("daemon-url")
		requirePR, _ := cmd.Flags().GetBool("require-pr")
		requireApproval, _ := cmd.Flags().GetBool("require-approval")
		noForcePush, _ := cmd.Flags().GetBool("no-force-push")

		client := cli.NewClient(daemonURL)
		body := map[string]bool{
			"require_pr":       requirePR,
			"require_approval": requireApproval,
			"no_force_push":    noForcePush,
		}
		var result map[string]interface{}
		if err := client.Put(fmt.Sprintf("/api/v1/repos/%s/protections/%s", repo, branch), body, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Protection rules set for %s/%s\n", repo, branch)
		fmt.Printf("  Require PR:       %v\n", result["require_pr"])
		fmt.Printf("  Require Approval: %v\n", result["require_approval"])
		fmt.Printf("  No Force Push:    %v\n", result["no_force_push"])
	},
}

var protectionRemoveCmd = &cobra.Command{
	Use:   "remove [repo] [branch]",
	Short: "Remove protection rules for a branch",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		branch := args[1]
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		if err := client.Delete(fmt.Sprintf("/api/v1/repos/%s/protections/%s", repo, branch)); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Protection rules removed for %s/%s\n", repo, branch)
	},
}

func init() {
	for _, c := range []*cobra.Command{protectionShowCmd, protectionSetCmd, protectionRemoveCmd} {
		c.Flags().String("daemon-url", "", "Daemon URL (default: http://localhost:7777)")
	}
	protectionSetCmd.Flags().Bool("require-pr", false, "Require pull request before merging")
	protectionSetCmd.Flags().Bool("require-approval", false, "Require approval before merging")
	protectionSetCmd.Flags().Bool("no-force-push", false, "Disallow force pushes")

	protectionCmd.AddCommand(protectionShowCmd)
	protectionCmd.AddCommand(protectionSetCmd)
	protectionCmd.AddCommand(protectionRemoveCmd)
	rootCmd.AddCommand(protectionCmd)
}
