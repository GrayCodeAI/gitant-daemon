package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lakshmanpatel/gitant/internal/api"
	"github.com/lakshmanpatel/gitant/internal/cli"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
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
		if port == 7777 {
			if envPort := os.Getenv("GITANT_PORT"); envPort != "" {
				fmt.Sscanf(envPort, "%d", &port)
			}
		}
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
		dataStoreDir := filepath.Join(dataDir, "data")
		if err := os.MkdirAll(dataStoreDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating data directory: %v\n", err)
			os.Exit(1)
		}

		repos, err := storage.NewRepositoryRegistry(reposDir, dataStoreDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating repository registry: %v\n", err)
			os.Exit(1)
		}

		// Create stores with persistence
		issueStore := crdt.NewIssueStore(filepath.Join(dataStoreDir, "issues.json"))
		prStore := crdt.NewPullRequestStore(filepath.Join(dataStoreDir, "prs.json"))
		blockstore := storage.NewBlockstore(filepath.Join(dataStoreDir, "blockstore.json"), filepath.Join(dataStoreDir, "blocks"))
		labelStore := crdt.NewLabelStore(dataStoreDir)
		taskStore := crdt.NewTaskStore(dataStoreDir)
		protectionStore := storage.NewProtectionStore(dataStoreDir)

		// Load persisted data
		if err := issueStore.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load issues: %v\n", err)
		}
		if err := prStore.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load PRs: %v\n", err)
		}
		if err := blockstore.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load blockstore: %v\n", err)
		}
		if err := labelStore.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load labels: %v\n", err)
		}
		if err := taskStore.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load tasks: %v\n", err)
		}
		if err := protectionStore.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load protections: %v\n", err)
		}
		fmt.Printf("Loaded %d blocks from disk\n", blockstore.Size())

		// Create and load webhook manager
		webhookManager := webhooks.NewManager()
		if err := webhookManager.Load(dataStoreDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load webhooks: %v\n", err)
		}

		// Create revocation store
		revocationStore := identity.NewRevocationStore(dataStoreDir)
		if err := revocationStore.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load revocations: %v\n", err)
		}

		// Create server
		server := api.NewServer(port, id, repos, issueStore, prStore, blockstore, labelStore, taskStore, protectionStore, webhookManager, revocationStore, dataStoreDir)

		// Start server in a goroutine
		errCh := make(chan error, 1)
		go func() {
			errCh <- server.Start()
		}()

		// Wait for interrupt signal or server error
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-sigCh:
			fmt.Printf("\nReceived signal: %v\n", sig)
		case err := <-errCh:
			if err != nil {
				fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Graceful shutdown with 10s timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Shutdown error: %v\n", err)
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
	Short: "Push changes to a remote gitant node",
	Run: func(cmd *cobra.Command, args []string) {
		remote, _ := cmd.Flags().GetString("remote")
		repo, _ := cmd.Flags().GetString("repo")
		if err := cli.Push(".", remote, repo); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull changes from a remote gitant node",
	Run: func(cmd *cobra.Command, args []string) {
		remote, _ := cmd.Flags().GetString("remote")
		repo, _ := cmd.Flags().GetString("repo")
		if err := cli.Pull(".", remote, repo); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var cloneCmd = &cobra.Command{
	Use:   "clone [repo-id] [directory]",
	Short: "Clone a repository from a gitant node",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repoID := args[0]
		dir := repoID
		if len(args) > 1 {
			dir = args[1]
		}
		remote, _ := cmd.Flags().GetString("remote")
		if err := cli.Clone(remote, repoID, dir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	serveCmd.Flags().IntP("port", "p", 7777, "Port to listen on")
	serveCmd.Flags().StringP("data-dir", "d", "", "Data directory (default: ~/.gitant)")

	pushCmd.Flags().StringP("remote", "r", "http://localhost:7777", "Remote daemon URL")
	pushCmd.Flags().String("repo", "", "Repository name (required)")
	pushCmd.MarkFlagRequired("repo")
	pullCmd.Flags().StringP("remote", "r", "http://localhost:7777", "Remote daemon URL")
	pullCmd.Flags().String("repo", "", "Repository name (required)")
	pullCmd.MarkFlagRequired("repo")
	cloneCmd.Flags().StringP("remote", "r", "http://localhost:7777", "Remote daemon URL")

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
