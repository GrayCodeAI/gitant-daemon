// git-remote-gitant is a git remote helper for the gitant:// URL scheme.
//
// Usage: git clone gitant://<repo-id> --remote http://localhost:7777
//
// Git automatically invokes this binary when it encounters a gitant:// URL.
// The helper communicates with the gitant daemon over HTTP to fetch/push objects.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/lakshmanpatel/gitant/internal/cli"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: git-remote-gitant <name> <url>\n")
		os.Exit(1)
	}

	remoteName := os.Args[1]
	repoID := os.Args[2]

	// Parse gitant://<repo-id> URL
	repoID = strings.TrimPrefix(repoID, "gitant://")
	repoID = strings.TrimPrefix(repoID, "gitant:")

	// Get daemon URL from env or default
	daemonURL := os.Getenv("GITANT_DAEMON_URL")
	if daemonURL == "" {
		daemonURL = "http://localhost:7777"
	}

	client := cli.NewClient(daemonURL)
	scanner := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case line == "capabilities":
			fmt.Fprintln(writer, "fetch")
			fmt.Fprintln(writer, "push")
			fmt.Fprintln(writer)
			writer.Flush()

		case line == "":
			writer.Flush()
			return

		case line == "list" || line == "list for-push":
			var result struct {
				Refs []struct {
					Name string `json:"name"`
					Hash string `json:"hash"`
				} `json:"refs"`
			}
			if err := client.Get(fmt.Sprintf("/api/v1/repos/%s/refs", repoID), &result); err != nil {
				fmt.Fprintf(os.Stderr, "error listing refs: %v\n", err)
				fmt.Fprintln(writer)
				writer.Flush()
				continue
			}
			for _, ref := range result.Refs {
				fmt.Fprintf(writer, "%s %s\n", ref.Hash, ref.Name)
			}
			fmt.Fprintln(writer)
			writer.Flush()

		case strings.HasPrefix(line, "fetch "):
			// fetch <sha1> <ref>
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			sha1 := parts[1]
			ref := parts[2]
			_ = remoteName

			// Fetch the object from daemon
			var obj struct {
				Hash    string `json:"hash"`
				Type    string `json:"type"`
				Content []byte `json:"content"`
			}
			if err := client.Get(fmt.Sprintf("/api/v1/repos/%s/objects/%s", repoID, sha1), &obj); err != nil {
				fmt.Fprintf(os.Stderr, "error fetching object %s: %v\n", sha1, err)
			} else {
				// Write the object data to stdout so git can consume it
				writer.Write(obj.Content)
				fmt.Fprintf(os.Stderr, "fetch %s %s (%d bytes)\n", sha1, ref, len(obj.Content))
			}
			fmt.Fprintln(writer)
			writer.Flush()

		case strings.HasPrefix(line, "push "):
			// push <refspec>
			refspec := strings.TrimPrefix(line, "push ")
			_ = remoteName

			// Parse refspec: +refs/heads/main:refs/heads/main
			refspec = strings.TrimPrefix(refspec, "+")
			parts := strings.SplitN(refspec, ":", 2)
			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "invalid refspec: %s\n", refspec)
				fmt.Fprintln(writer)
				writer.Flush()
				continue
			}

			localRef := parts[0]
			remoteRef := parts[1]

			// Use CLI push
			fmt.Fprintf(os.Stderr, "push %s -> %s (using CLI push)\n", localRef, remoteRef)
			if err := cli.Push(".", daemonURL, repoID); err != nil {
				fmt.Fprintf(os.Stderr, "push error: %v\n", err)
			}
			fmt.Fprintln(writer)
			writer.Flush()
		}
	}
}
