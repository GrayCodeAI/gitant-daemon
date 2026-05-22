package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/spf13/cobra"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Interactive setup wizard for gitant",
	Run: func(cmd *cobra.Command, args []string) {
		nonInteractive, _ := cmd.Flags().GetBool("yes")
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("gitant quickstart")
		fmt.Println("=================")
		fmt.Println("This wizard will set up gitant on your machine.")
		fmt.Println()

		// 1. Data directory
		home, _ := os.UserHomeDir()
		defaultDataDir := filepath.Join(home, ".gitant")
		dataDir := defaultDataDir

		if !nonInteractive {
			fmt.Printf("Data directory [%s]: ", defaultDataDir)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				dataDir = input
			}
		}

		if err := os.MkdirAll(dataDir, 0700); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating data directory: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[OK] Data directory: %s\n", dataDir)

		// 2. Generate identity
		identityPath := filepath.Join(dataDir, "identity.key")
		if _, err := os.Stat(identityPath); err == nil {
			fmt.Println("[OK] Identity already exists")
		} else {
			id, err := identity.NewIdentity()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating identity: %v\n", err)
				os.Exit(1)
			}
			if err := id.Save(identityPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving identity: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("[OK] Generated identity: %s\n", id.DID)
		}

		// 3. Create directories
		for _, dir := range []string{"repos", "data"} {
			path := filepath.Join(dataDir, dir)
			if err := os.MkdirAll(path, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", dir, err)
				os.Exit(1)
			}
			fmt.Printf("[OK] Created %s\n", dir)
		}

		// 4. Port
		port := "7777"
		if !nonInteractive {
			fmt.Printf("Daemon port [7777]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				port = input
			}
		}
		fmt.Printf("[OK] Port: %s\n", port)

		// Summary
		fmt.Println()
		fmt.Println("=================")
		fmt.Println("Setup complete!")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Printf("  1. Start the daemon:  gitant serve --port %s\n", port)
		fmt.Println("  2. Create a repo:     gitant repo list")
		fmt.Println("  3. Open the web UI:   http://localhost:3303")
		fmt.Println()
	},
}

func init() {
	quickstartCmd.Flags().BoolP("yes", "y", false, "Non-interactive mode (use defaults)")
	rootCmd.AddCommand(quickstartCmd)
}
