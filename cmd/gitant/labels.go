package main

import (
	"fmt"
	"os"

	"github.com/lakshmanpatel/gitant/internal/cli"
	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Manage repository labels",
}

var labelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List labels for a repository",
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		client := cli.NewClient(daemonURL)
		var result struct {
			Labels []struct {
				Name  string `json:"name"`
				Color string `json:"color"`
			} `json:"labels"`
			Total int `json:"total"`
		}
		if err := client.Get(fmt.Sprintf("/api/v1/repos/%s/labels", repo), &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		for _, l := range result.Labels {
			fmt.Printf("%s\t%s\n", l.Name, l.Color)
		}
		fmt.Fprintf(os.Stderr, "%d label(s)\n", result.Total)
	},
}

var labelCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a label for a repository",
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		name, _ := cmd.Flags().GetString("name")
		color, _ := cmd.Flags().GetString("color")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required")
			os.Exit(1)
		}

		client := cli.NewClient(daemonURL)
		var result map[string]interface{}
		if err := client.Post(fmt.Sprintf("/api/v1/repos/%s/labels", repo), map[string]string{"name": name, "color": color}, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created label: %s (%s)\n", name, color)
	},
}

var labelDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a label from a repository",
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		name, _ := cmd.Flags().GetString("name")
		daemonURL, _ := cmd.Flags().GetString("daemon-url")

		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required")
			os.Exit(1)
		}

		client := cli.NewClient(daemonURL)
		if err := client.Delete(fmt.Sprintf("/api/v1/repos/%s/labels/%s", repo, name)); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted label: %s\n", name)
	},
}

func init() {
	for _, c := range []*cobra.Command{labelListCmd, labelCreateCmd, labelDeleteCmd} {
		c.Flags().StringP("repo", "r", "", "Repository name (required)")
		c.MarkFlagRequired("repo")
		c.Flags().String("daemon-url", "", "Daemon URL (default: http://localhost:7777)")
	}
	labelCreateCmd.Flags().StringP("name", "n", "", "Label name (required)")
	labelCreateCmd.Flags().StringP("color", "c", "#6b7280", "Label color (hex)")
	labelDeleteCmd.Flags().StringP("name", "n", "", "Label name (required)")

	labelCmd.AddCommand(labelListCmd, labelCreateCmd, labelDeleteCmd)
	rootCmd.AddCommand(labelCmd)
}
