package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lakshmanpatel/gitant/internal/api"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gitant",
	Short: "Decentralized GitHub for solo developers",
	Long:  "gitant is a decentralized git hosting platform for solo developers and AI agents.",
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the gitant daemon",
	Long:  "Start the gitant daemon with P2P networking, IPFS storage, and HTTP API.",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		dataDir, _ := cmd.Flags().GetString("data-dir")

		if dataDir == "" {
			home, _ := os.UserHomeDir()
			dataDir = filepath.Join(home, ".gitant")
		}

		fmt.Printf("Starting gitant daemon on port %d...\n", port)
		fmt.Printf("Data directory: %s\n", dataDir)

		// Load or create identity
		identityPath := filepath.Join(dataDir, "identity.key")
		var id *identity.Identity
		var err error

		if _, statErr := os.Stat(identityPath); os.IsNotExist(statErr) {
			fmt.Println("Creating new identity...")
			id, err = identity.NewIdentity()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating identity: %v\n", err)
				os.Exit(1)
			}
			if err := id.Save(identityPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving identity: %v\n", err)
				os.Exit(1)
			}
		} else {
			id, err = identity.LoadIdentity(identityPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading identity: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Printf("Identity: %s\n", id.DID)

		// Create repository registry
		reposDir := filepath.Join(dataDir, "repos")
		repos, err := storage.NewRepositoryRegistry(reposDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating repository registry: %v\n", err)
			os.Exit(1)
		}

		// Create stores
		issueStore := crdt.NewIssueStore()
		prStore := crdt.NewPullRequestStore()
		blockstore := storage.NewBlockstore()

		// Create and start server
		server := api.NewServer(port, id, repos, issueStore, prStore, blockstore)
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
			os.Exit(1)
		}
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new repository",
	Long:  "Initialize a new gitant repository in the current directory.",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
			os.Exit(1)
		}

		if _, err := storage.InitRepository(cwd); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing repository: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Initialized gitant repository in %s\n", cwd)
	},
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push objects to remote",
	Long:  "Push git objects to a remote gitant node.",
	Run: func(cmd *cobra.Command, args []string) {
		remote, _ := cmd.Flags().GetString("remote")
		fmt.Printf("Pushing to %s...\n", remote)
		fmt.Println("TODO: Implement push")
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull objects from remote",
	Long:  "Pull git objects from a remote gitant node.",
	Run: func(cmd *cobra.Command, args []string) {
		remote, _ := cmd.Flags().GetString("remote")
		fmt.Printf("Pulling from %s...\n", remote)
		fmt.Println("TODO: Implement pull")
	},
}

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone a repository",
	Long:  "Clone a repository from a remote gitant node.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Usage: gitant clone <remote>\n")
			os.Exit(1)
		}
		remote := args[0]
		fmt.Printf("Cloning from %s...\n", remote)
		fmt.Println("TODO: Implement clone")
	},
}

func init() {
	serveCmd.Flags().IntP("port", "p", 7777, "Port to listen on")
	serveCmd.Flags().StringP("data-dir", "d", "", "Data directory (default: ~/.gitant)")

	pushCmd.Flags().StringP("remote", "r", "", "Remote to push to")
	pullCmd.Flags().StringP("remote", "r", "", "Remote to pull from")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(cloneCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
