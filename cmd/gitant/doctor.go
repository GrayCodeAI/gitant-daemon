package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check gitant installation and connectivity",
	Run: func(cmd *cobra.Command, args []string) {
		daemonURL, _ := cmd.Flags().GetString("daemon-url")
		dataDir, _ := cmd.Flags().GetString("data-dir")

		if dataDir == "" {
			home, _ := os.UserHomeDir()
			dataDir = filepath.Join(home, ".gitant")
		}

		passed, warned, failed := 0, 0, 0

		check := func(name string, ok bool, msg string) {
			if ok {
				fmt.Printf("  [OK]   %s\n", name)
				passed++
			} else {
				fmt.Printf("  [FAIL] %s: %s\n", name, msg)
				failed++
			}
		}

		warn := func(name string, ok bool, msg string) {
			if ok {
				fmt.Printf("  [OK]   %s\n", name)
				passed++
			} else {
				fmt.Printf("  [WARN] %s: %s\n", name, msg)
				warned++
			}
		}

		fmt.Println("gitant doctor")
		fmt.Println("=============")

		// 1. Daemon connectivity
		fmt.Println("\nDaemon:")
		resp, err := http.Get(daemonURL + "/health")
		check("Daemon reachable", err == nil && resp != nil && resp.StatusCode == 200,
			func() string {
				if err != nil {
					return err.Error()
				}
				return fmt.Sprintf("status %d", resp.StatusCode)
			}())

		// 2. Identity
		fmt.Println("\nIdentity:")
		identityPath := filepath.Join(dataDir, "identity.key")
		_, err = os.Stat(identityPath)
		check("Identity file exists", err == nil, identityPath)

		if err == nil {
			id, err := identity.LoadIdentity(identityPath)
			check("Identity valid", err == nil, "failed to load")
			if err == nil {
				check("DID generated", id.DID != "", "empty DID")
			}
		}

		// 3. Data directory
		fmt.Println("\nData directory:")
		_, err = os.Stat(dataDir)
		check("Data directory exists", err == nil, dataDir)

		reposDir := filepath.Join(dataDir, "repos")
		_, err = os.Stat(reposDir)
		warn("Repos directory", err == nil, reposDir)

		dataStoreDir := filepath.Join(dataDir, "data")
		_, err = os.Stat(dataStoreDir)
		warn("Data store directory", err == nil, dataStoreDir)

		// 4. JSON files parseable
		fmt.Println("\nData files:")
		for _, file := range []string{"issues.json", "prs.json", "registry.json", "labels.json", "tasks.json"} {
			path := filepath.Join(dataStoreDir, file)
			data, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				fmt.Printf("  [OK]   %s (not created yet)\n", file)
				passed++
				continue
			}
			if err != nil {
				fmt.Printf("  [FAIL] %s: %s\n", file, err.Error())
				failed++
				continue
			}
			var js json.RawMessage
			err = json.Unmarshal(data, &js)
			check(file+" parseable", err == nil, "invalid JSON")
		}

		// 5. Git binary
		fmt.Println("\nSystem:")
		_, err = exec.LookPath("git")
		warn("git binary", err == nil, "git not found in PATH")

		// Summary
		fmt.Printf("\n=============\n")
		fmt.Printf("Results: %d passed, %d warnings, %d failed\n", passed, warned, failed)
		if failed > 0 {
			os.Exit(1)
		}
	},
}

func init() {
	doctorCmd.Flags().String("daemon-url", "http://localhost:7777", "Daemon URL")
	doctorCmd.Flags().String("data-dir", "", "Data directory (default: ~/.gitant)")
	rootCmd.AddCommand(doctorCmd)
}
