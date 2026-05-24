package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/lakshmanpatel/gitant/internal/cli"
	"github.com/spf13/cobra"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage issues",
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues in a repository",
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		status, _ := cmd.Flags().GetString("status")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		path := fmt.Sprintf("/api/v1/repos/%s/issues", repo)
		if status != "" {
			path += "?status=" + status
		}

		var result struct {
			Issues []struct {
				ID     string `json:"id"`
				Title  string `json:"title"`
				Status string `json:"status"`
				Author string `json:"author"`
			} `json:"issues"`
			Total int `json:"total"`
		}
		if err := client.Get(path, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		for _, issue := range result.Issues {
			fmt.Printf("%s\t%s\t[%s]\t%s\n", issue.ID, issue.Status, issue.Author, issue.Title)
		}
		fmt.Fprintf(os.Stderr, "%d issue(s)\n", result.Total)
	},
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")
		labels, _ := cmd.Flags().GetString("labels")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		if title == "" {
			fmt.Fprintln(os.Stderr, "Error: --title is required")
			os.Exit(1)
		}

		client := cli.NewClient(daemonURL)
		req := map[string]interface{}{
			"title": title,
			"body":  body,
		}
		if labels != "" {
			req["labels"] = strings.Split(labels, ",")
		}

		var result map[string]interface{}
		if err := client.Post(fmt.Sprintf("/api/v1/repos/%s/issues", repo), req, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created issue: %s\n", result["id"])
	},
}

var issueCloseCmd = &cobra.Command{
	Use:   "close [issue-id]",
	Short: "Close an issue",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Post(fmt.Sprintf("/api/v1/repos/%s/issues/%s/close", repo, args[0]), nil, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Closed issue %s\n", args[0])
	},
}

var issueCommentCmd = &cobra.Command{
	Use:   "comment [issue-id]",
	Short: "Comment on an issue",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		body, _ := cmd.Flags().GetString("body")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		if body == "" {
			fmt.Fprintln(os.Stderr, "Error: --body is required")
			os.Exit(1)
		}

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Post(fmt.Sprintf("/api/v1/repos/%s/issues/%s/comment", repo, args[0]), map[string]string{"body": body}, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Commented on issue %s\n", args[0])
	},
}

var issueCommentsCmd = &cobra.Command{
	Use:   "comments [issue-id]",
	Short: "List comments on an issue",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result struct {
			Comments []struct {
				ID        string `json:"id"`
				Author    string `json:"author"`
				Body      string `json:"body"`
				Timestamp string `json:"timestamp"`
			} `json:"comments"`
			Total int `json:"total"`
		}
		if err := client.Get(fmt.Sprintf("/api/v1/repos/%s/issues/%s/comments", repo, args[0]), &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		for _, c := range result.Comments {
			fmt.Printf("%s\t%s\t%s\t%s\n", c.ID, c.Author, c.Timestamp, c.Body)
		}
		fmt.Fprintf(os.Stderr, "%d comment(s)\n", result.Total)
	},
}

func init() {
	for _, c := range []*cobra.Command{issueListCmd, issueCreateCmd, issueCloseCmd, issueCommentCmd, issueCommentsCmd} {
		c.Flags().StringP("repo", "r", "", "Repository name (required)")
		c.MarkFlagRequired("repo")
		c.Flags().String("daemon-url", "", "Daemon URL (default: http://localhost:7777)")
	}
	issueListCmd.Flags().String("status", "", "Filter by status (open|closed)")
	issueCreateCmd.Flags().StringP("title", "t", "", "Issue title (required)")
	issueCreateCmd.Flags().StringP("body", "b", "", "Issue body")
	issueCreateCmd.Flags().StringP("labels", "l", "", "Comma-separated labels")
	issueCommentCmd.Flags().StringP("body", "b", "", "Comment body (required)")

	issueCmd.AddCommand(issueListCmd, issueCreateCmd, issueCloseCmd, issueCommentCmd, issueCommentsCmd)
	rootCmd.AddCommand(issueCmd)
}
