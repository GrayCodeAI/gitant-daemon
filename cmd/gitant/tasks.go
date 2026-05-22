package main

import (
	"fmt"
	"os"

	"github.com/lakshmanpatel/gitant/internal/cli"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage agent tasks",
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks for a repository",
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		status, _ := cmd.Flags().GetString("status")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		path := fmt.Sprintf("/api/v1/repos/%s/tasks", repo)
		if status != "" {
			path += "?status=" + status
		}

		var result struct {
			Tasks []struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				Status      string `json:"status"`
				ClaimedBy   string `json:"claimed_by"`
				CreatedBy   string `json:"created_by"`
				Description string `json:"description"`
			} `json:"tasks"`
			Total int `json:"total"`
		}
		if err := client.Get(path, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		for _, t := range result.Tasks {
			claimed := ""
			if t.ClaimedBy != "" {
				claimed = fmt.Sprintf(" (%s)", t.ClaimedBy)
			}
			fmt.Printf("%s\t[%s]%s\t%s\n", t.ID, t.Status, claimed, t.Title)
		}
		fmt.Fprintf(os.Stderr, "%d task(s)\n", result.Total)
	},
}

var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		if title == "" {
			fmt.Fprintln(os.Stderr, "Error: --title is required")
			os.Exit(1)
		}

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Post(fmt.Sprintf("/api/v1/repos/%s/tasks", repo), map[string]string{"title": title, "description": description}, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created task: %s\n", result["id"])
	},
}

var taskClaimCmd = &cobra.Command{
	Use:   "claim [task-id]",
	Short: "Claim a task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Post(fmt.Sprintf("/api/v1/repos/%s/tasks/%s/claim", repo, args[0]), nil, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Claimed task %s\n", args[0])
	},
}

var taskCompleteCmd = &cobra.Command{
	Use:   "complete [task-id]",
	Short: "Complete a task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		result, _ := cmd.Flags().GetString("result")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var res map[string]interface{}
		if err := client.Post(fmt.Sprintf("/api/v1/repos/%s/tasks/%s/complete", repo, args[0]), map[string]string{"result": result}, &res); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Completed task %s\n", args[0])
	},
}

func init() {
	for _, c := range []*cobra.Command{taskListCmd, taskCreateCmd, taskClaimCmd, taskCompleteCmd} {
		c.Flags().StringP("repo", "r", "", "Repository name (required)")
		c.MarkFlagRequired("repo")
		c.Flags().String("daemon-url", "", "Daemon URL (default: http://localhost:7777)")
	}
	taskCreateCmd.Flags().StringP("title", "t", "", "Task title (required)")
	taskCreateCmd.Flags().StringP("description", "d", "", "Task description")
	taskCompleteCmd.Flags().String("result", "", "Task result")

	taskCmd.AddCommand(taskListCmd, taskCreateCmd, taskClaimCmd, taskCompleteCmd)
	rootCmd.AddCommand(taskCmd)
}
