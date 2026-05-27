package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lakshmanpatel/gitant/internal/api"
	"github.com/lakshmanpatel/gitant/internal/cli"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/lakshmanpatel/gitant/internal/ipfs"
	"github.com/lakshmanpatel/gitant/internal/network"
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
		// Set up structured logging
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		slog.SetDefault(logger)

		port, _ := cmd.Flags().GetInt("port")
		if port == 7777 {
			if envPort := os.Getenv("GITANT_PORT"); envPort != "" {
				if _, err := fmt.Sscanf(envPort, "%d", &port); err != nil {
					slog.Warn("invalid GITANT_PORT, using default", "value", envPort, "error", err)
				}
			}
		}
		dataDir, _ := cmd.Flags().GetString("data-dir")

		if dataDir == "" {
			home, _ := os.UserHomeDir()
			dataDir = filepath.Join(home, ".gitant")
		}

		slog.Info("starting gitant daemon", "port", port)
		slog.Info("data directory", "path", dataDir)

		// Load or create identity
		identityPath := filepath.Join(dataDir, "identity.key")
		var id *identity.Identity
		var err error

		if _, statErr := os.Stat(identityPath); os.IsNotExist(statErr) {
			slog.Info("creating new identity")
			id, err = identity.NewIdentity()
			if err != nil {
				slog.Error("failed to create identity", "error", err)
				os.Exit(1)
			}
			if err := id.Save(identityPath); err != nil {
				slog.Error("failed to save identity", "error", err)
				os.Exit(1)
			}
		} else {
			id, err = identity.LoadIdentity(identityPath)
			if err != nil {
				slog.Error("failed to load identity", "error", err)
				os.Exit(1)
			}
		}
		slog.Info("identity loaded", "did", id.DID)

		// Create repository registry
		reposDir := filepath.Join(dataDir, "repos")
		dataStoreDir := filepath.Join(dataDir, "data")
		if err := os.MkdirAll(dataStoreDir, 0755); err != nil {
			slog.Error("failed to create data directory", "error", err)
			os.Exit(1)
		}

		repos, err := storage.NewRepositoryRegistry(reposDir, dataStoreDir)
		if err != nil {
			slog.Error("failed to create repository registry", "error", err)
			os.Exit(1)
		}

		// Create stores with persistence
		issueStore := crdt.NewIssueStore(filepath.Join(dataStoreDir, "issues.json"))
		prStore := crdt.NewPullRequestStore(filepath.Join(dataStoreDir, "prs.json"))
		blockstore := storage.NewBlockstore(filepath.Join(dataStoreDir, "blockstore.json"), filepath.Join(dataStoreDir, "blocks"))
		labelStore := crdt.NewLabelStore(dataStoreDir)
		taskStore := crdt.NewTaskStore(dataStoreDir)
		releaseStore := crdt.NewReleaseStore(dataStoreDir)
		protectionStore := storage.NewProtectionStore(dataStoreDir)

		// Load persisted data
		if err := issueStore.Load(); err != nil {
			slog.Warn("failed to load issues", "error", err)
		}
		if err := prStore.Load(); err != nil {
			slog.Warn("failed to load PRs", "error", err)
		}
		if err := blockstore.Load(); err != nil {
			slog.Warn("failed to load blockstore", "error", err)
		}
		if err := labelStore.Load(); err != nil {
			slog.Warn("failed to load labels", "error", err)
		}
		if err := taskStore.Load(); err != nil {
			slog.Warn("failed to load tasks", "error", err)
		}
		if err := protectionStore.Load(); err != nil {
			slog.Warn("failed to load protections", "error", err)
		}
		if err := releaseStore.Load(); err != nil {
			slog.Warn("failed to load releases", "error", err)
		}
		slog.Info("blocks loaded from disk", "count", blockstore.Size())

		// Create and load webhook manager
		webhookManager := webhooks.NewManager()
		if err := webhookManager.Load(dataStoreDir); err != nil {
			slog.Warn("failed to load webhooks", "error", err)
		}

		// Create revocation store
		revocationStore := identity.NewRevocationStore(dataStoreDir)
		if err := revocationStore.Load(); err != nil {
			slog.Warn("failed to load revocations", "error", err)
		}

		// Parse CORS origins from environment
		var corsOrigins []string
		if envOrigins := os.Getenv("GITANT_CORS_ORIGINS"); envOrigins != "" {
			for _, o := range strings.Split(envOrigins, ",") {
				if trimmed := strings.TrimSpace(o); trimmed != "" {
					corsOrigins = append(corsOrigins, trimmed)
				}
			}
		}

		// Create server
		server := api.NewServer(port, id, repos, issueStore, prStore, blockstore, labelStore, taskStore, releaseStore, protectionStore, webhookManager, revocationStore, dataStoreDir, corsOrigins)

		p2pEnabled, _ := cmd.Flags().GetBool("p2p")
		if envP2P := os.Getenv("GITANT_P2P"); envP2P != "" {
			p2pEnabled = envP2P == "1" || strings.EqualFold(envP2P, "true")
		}
		if p2pEnabled {
			p2pListen, _ := cmd.Flags().GetString("p2p-listen")
			p2pMDNS, _ := cmd.Flags().GetBool("p2p-mdns")
			bootstrapPeers, _ := cmd.Flags().GetStringSlice("bootstrap-peers")
			if envBootstrap := os.Getenv("GITANT_BOOTSTRAP_PEERS"); envBootstrap != "" {
				bootstrapPeers = append(bootstrapPeers, strings.Split(envBootstrap, ",")...)
			}
			bootstrapPeers = network.MergeBootstrapPeers(bootstrapPeers)

			netNode, err := network.StartNode(context.Background(), network.NodeConfig{
				ListenAddr:     p2pListen,
				EnableMDNS:     p2pMDNS,
				BootstrapPeers: bootstrapPeers,
				ServerDID:      id.DID,
				HTTPPort:       port,
			})
			if err != nil {
				slog.Warn("P2P startup failed, continuing HTTP-only", "error", err)
			} else {
				var pinner network.ObjectPinner
				ipfsPin, _ := cmd.Flags().GetBool("ipfs-pin")
				if envIPFS := os.Getenv("GITANT_IPFS_PIN"); envIPFS != "" {
					ipfsPin = envIPFS == "1" || strings.EqualFold(envIPFS, "true")
				}
				if ipfsPin {
					pinner = ipfs.NewPinningAdapter(ipfs.NewPinningStore())
					slog.Info("IPFS warm pinning enabled")
				}
				server.SetNetwork(netNode, pinner)
			}
		}

		tlsCert, _ := cmd.Flags().GetString("tls-cert")
		tlsKey, _ := cmd.Flags().GetString("tls-key")

		// Start server in a goroutine
		errCh := make(chan error, 1)
		go func() {
			errCh <- server.Start(tlsCert, tlsKey)
		}()

		// Wait for interrupt signal or server error
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-sigCh:
			slog.Info("received shutdown signal", "signal", sig)
		case err := <-errCh:
			if err != nil {
				slog.Error("server error", "error", err)
				os.Exit(1)
			}
			return
		}

		// Graceful shutdown with 10s timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", "error", err)
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
			slog.Error("failed to get current directory", "error", err)
			os.Exit(1)
		}

		if _, err := storage.InitRepository(cwd); err != nil {
			slog.Error("failed to initialize repository", "error", err)
			os.Exit(1)
		}

		slog.Info("initialized gitant repository", "path", cwd)
	},
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push changes to a remote gitant node",
	Run: func(cmd *cobra.Command, args []string) {
		remote, _ := cmd.Flags().GetString("remote")
		repo, _ := cmd.Flags().GetString("repo")
		if err := cli.Push(".", remote, repo); err != nil {
			slog.Error("push failed", "error", err)
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
			slog.Error("pull failed", "error", err)
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
			slog.Error("clone failed", "error", err)
			os.Exit(1)
		}
	},
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup gitant data directory",
	Long:  "Create a timestamped backup of the gitant data directory (JSON stores, identity, revocations).",
	Run: func(cmd *cobra.Command, args []string) {
		dataDir, _ := cmd.Flags().GetString("data-dir")
		outputDir, _ := cmd.Flags().GetString("output")

		if dataDir == "" {
			home, _ := os.UserHomeDir()
			dataDir = filepath.Join(home, ".gitant")
		}

		timestamp := time.Now().Format("20060102-150405")
		backupDir := filepath.Join(outputDir, "gitant-backup-"+timestamp)

		if err := os.MkdirAll(backupDir, 0755); err != nil {
			slog.Error("failed to create backup directory", "error", err)
			os.Exit(1)
		}

		// Backup the three top-level data units used by `gitant serve`.
		backupItems := []string{"identity.key", "repos", "data"}

		backedUp := 0
		for _, name := range backupItems {
			src := filepath.Join(dataDir, name)
			dst := filepath.Join(backupDir, name)

			info, err := os.Stat(src)
			if os.IsNotExist(err) {
				continue
			}

			if info.IsDir() {
				if err := copyDir(src, dst); err != nil {
					slog.Warn("failed to backup directory", "path", name, "error", err)
				} else {
					backedUp++
				}
			} else {
				if err := copyFile(src, dst); err != nil {
					slog.Warn("failed to backup file", "path", name, "error", err)
				} else {
					backedUp++
				}
			}
		}

		slog.Info("backup complete", "path", backupDir, "items", backedUp)
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore gitant data from backup",
	Long:  "Restore gitant data from a previously created backup directory. Existing data is NOT overwritten.",
	Run: func(cmd *cobra.Command, args []string) {
		dataDir, _ := cmd.Flags().GetString("data-dir")
		inputDir, _ := cmd.Flags().GetString("input")

		if dataDir == "" {
			home, _ := os.UserHomeDir()
			dataDir = filepath.Join(home, ".gitant")
		}

		if _, err := os.Stat(inputDir); os.IsNotExist(err) {
			slog.Error("backup directory not found", "path", inputDir)
			os.Exit(1)
		}

		if err := os.MkdirAll(dataDir, 0755); err != nil {
			slog.Error("failed to create data directory", "error", err)
			os.Exit(1)
		}

		entries, err := os.ReadDir(inputDir)
		if err != nil {
			slog.Error("failed to read backup directory", "error", err)
			os.Exit(1)
		}

		restored := 0
		for _, entry := range entries {
			src := filepath.Join(inputDir, entry.Name())
			dst := filepath.Join(dataDir, entry.Name())

			// Don't overwrite existing files
			if _, err := os.Stat(dst); err == nil {
				slog.Info("skipping (already exists)", "path", entry.Name())
				continue
			}

			if entry.IsDir() {
				if err := copyDir(src, dst); err != nil {
					slog.Warn("failed to restore directory", "path", entry.Name(), "error", err)
				} else {
					restored++
				}
			} else {
				if err := copyFile(src, dst); err != nil {
					slog.Warn("failed to restore file", "path", entry.Name(), "error", err)
				} else {
					restored++
				}
			}
		}

		slog.Info("restore complete", "items", restored)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gitant %s (commit %s, built %s)\n", api.Version, api.Commit, api.BuildTime)
	},
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, info.Mode())
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}

func init() {
	serveCmd.Flags().IntP("port", "p", 7777, "Port to listen on")
	serveCmd.Flags().StringP("data-dir", "d", "", "Data directory (default: ~/.gitant)")
	serveCmd.Flags().String("tls-cert", "", "TLS certificate file path")
	serveCmd.Flags().String("tls-key", "", "TLS private key file path")
	serveCmd.Flags().Bool("p2p", false, "Enable libp2p networking (DHT + GossipSub)")
	serveCmd.Flags().String("p2p-listen", "/ip4/0.0.0.0/tcp/0", "libp2p listen multiaddr")
	serveCmd.Flags().Bool("p2p-mdns", true, "Enable mDNS peer discovery on LAN")
	serveCmd.Flags().StringSlice("bootstrap-peers", nil, "Bootstrap peer multiaddrs (repeatable)")
	serveCmd.Flags().Bool("ipfs-pin", false, "Pin replicated git objects in warm IPFS storage")

	pushCmd.Flags().StringP("remote", "r", "http://localhost:7777", "Remote daemon URL")
	pushCmd.Flags().String("repo", "", "Repository name (required)")
	pushCmd.MarkFlagRequired("repo")
	pullCmd.Flags().StringP("remote", "r", "http://localhost:7777", "Remote daemon URL")
	pullCmd.Flags().String("repo", "", "Repository name (required)")
	pullCmd.MarkFlagRequired("repo")
	cloneCmd.Flags().StringP("remote", "r", "http://localhost:7777", "Remote daemon URL")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(cloneCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)

	backupCmd.Flags().StringP("output", "o", "", "Backup output directory (required)")
	backupCmd.Flags().StringP("data-dir", "d", "", "Data directory (default: ~/.gitant)")
	backupCmd.MarkFlagRequired("output")

	restoreCmd.Flags().StringP("input", "i", "", "Backup directory to restore from (required)")
	restoreCmd.Flags().StringP("data-dir", "d", "", "Data directory (default: ~/.gitant)")
	restoreCmd.MarkFlagRequired("input")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
