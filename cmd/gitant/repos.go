package main

import (
	"fmt"
	"os"

	"github.com/lakshmanpatel/gitant/internal/cli"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repositories",
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result struct {
			Repos []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				Stars       int    `json:"stars"`
			} `json:"repos"`
			Total int `json:"total"`
		}
		if err := client.Get("/api/v1/repos", &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		for _, repo := range result.Repos {
			fmt.Printf("%s\t%s\tstars=%d\t%s\n", repo.ID, repo.Name, repo.Stars, repo.Description)
		}
		fmt.Fprintf(os.Stderr, "%d repo(s)\n", result.Total)
	},
}

var repoStarCmd = &cobra.Command{
	Use:   "star [repo-id]",
	Short: "Star a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Post(fmt.Sprintf("/api/v1/repos/%s/star", args[0]), nil, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Starred %s (%v stars)\n", args[0], result["stars"])
	},
}

var repoUnstarCmd = &cobra.Command{
	Use:   "unstar [repo-id]",
	Short: "Unstar a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Post(fmt.Sprintf("/api/v1/repos/%s/unstar", args[0]), nil, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Unstarred %s (%v stars)\n", args[0], result["stars"])
	},
}

var repoForkCmd = &cobra.Command{
	Use:   "fork [source] [name]",
	Short: "Fork a repository",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		body := map[string]string{"name": args[1]}
		if err := client.Post(fmt.Sprintf("/api/v1/repos/%s/fork", args[0]), body, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Forked %s -> %s\n", args[0], result["id"])
		if forkedFrom, ok := result["forked_from"]; ok {
			fmt.Printf("Forked from: %s\n", forkedFrom)
		}
	},
}

func init() {
	for _, c := range []*cobra.Command{repoListCmd, repoStarCmd, repoUnstarCmd, repoForkCmd} {
		c.Flags().String("daemon-url", "", "Daemon URL (default: http://localhost:7777)")
	}

	repoCmd.AddCommand(repoListCmd, repoStarCmd, repoUnstarCmd, repoForkCmd)
	rootCmd.AddCommand(repoCmd)
}
