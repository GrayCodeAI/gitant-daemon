package transport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SSHServer handles SSH connections for git operations
type SSHServer struct {
	sshConfig *ssh.ServerConfig
	listener  net.Listener
	wg        sync.WaitGroup
	done      chan struct{}
	port      int
	userStore store.UserStore
}

// SSHConfig holds SSH server configuration
type SSHConfig struct {
	Port               int
	HostKeyPath        string
	AuthorizedKeysPath string
	UserStore          store.UserStore
}

// NewSSHServer creates a new SSH server
func NewSSHServer(config SSHConfig) (*SSHServer, error) {
	if config.Port == 0 {
		config.Port = 2222
	}

	server := &SSHServer{
		port:      config.Port,
		done:      make(chan struct{}),
		userStore: config.UserStore,
	}

	sshConfig := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			return nil, fmt.Errorf("password authentication not supported")
		},
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return server.authenticatePublicKey(c, key)
		},
	}

	server.sshConfig = sshConfig

	// Generate or load host key
	if err := setupHostKey(config.HostKeyPath, sshConfig); err != nil {
		return nil, fmt.Errorf("setting up host key: %w", err)
	}

	return server, nil
}

// authenticatePublicKey validates an SSH public key against registered user keys.
func (s *SSHServer) authenticatePublicKey(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	if s.userStore == nil {
		slog.Warn("SSH auth: no user store configured, rejecting key",
			"user", c.User(), "fingerprint", ssh.FingerprintSHA256(key))
		return nil, fmt.Errorf("SSH authentication not configured")
	}

	fingerprint := ssh.FingerprintSHA256(key)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, _, err := s.userStore.FindByFingerprint(ctx, fingerprint)
	if err != nil {
		slog.Warn("SSH auth: unknown key",
			"user", c.User(), "fingerprint", fingerprint)
		return nil, fmt.Errorf("unknown SSH key")
	}

	slog.Info("SSH public key auth", "user", user.Username, "fingerprint", fingerprint)

	// Store the authenticated user ID in permissions extensions for downstream use
	return &ssh.Permissions{
		Extensions: map[string]string{
			"user-id":  user.ID,
			"username": user.Username,
		},
	}, nil
}

// setupHostKey generates or loads the SSH host key
func setupHostKey(hostKeyPath string, config *ssh.ServerConfig) error {
	if hostKeyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		hostKeyPath = filepath.Join(home, ".gitant", "ssh_host_key")
		if err := os.MkdirAll(filepath.Dir(hostKeyPath), 0700); err != nil {
			return fmt.Errorf("creating host key directory: %w", err)
		}
	}

	// Check if key exists
	if _, err := os.Stat(hostKeyPath); os.IsNotExist(err) {
		// Generate new key
		slog.Info("Generating new SSH host key", "path", hostKeyPath)
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return fmt.Errorf("failed to generate RSA key: %w", err)
		}

		// Encode to PEM
		privateKeyPEM := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}

		// Write to file
		keyFile, err := os.Create(hostKeyPath)
		if err != nil {
			return fmt.Errorf("failed to create key file: %w", err)
		}
		defer keyFile.Close()

		if err := pem.Encode(keyFile, privateKeyPEM); err != nil {
			return fmt.Errorf("failed to encode private key: %w", err)
		}

		// Set proper permissions
		if err := os.Chmod(hostKeyPath, 0600); err != nil {
			return fmt.Errorf("failed to set key permissions: %w", err)
		}

		// Create signer from private key
		signer, err := ssh.NewSignerFromKey(privateKey)
		if err != nil {
			return fmt.Errorf("failed to create signer: %w", err)
		}

		config.AddHostKey(signer)
	} else {
		// Load existing key
		keyData, err := os.ReadFile(hostKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}

		config.AddHostKey(signer)
	}

	return nil
}

// Start starts the SSH server
func (s *SSHServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on SSH port: %w", err)
	}
	s.listener = listener

	slog.Info("SSH server listening", "port", s.port)

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// acceptConnections accepts incoming SSH connections
func (s *SSHServer) acceptConnections() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				slog.Error("SSH accept error", "error", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles an individual SSH connection
func (s *SSHServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Upgrade to SSH connection
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.sshConfig)
	if err != nil {
		slog.Error("SSH handshake failed", "error", err)
		return
	}
	defer sshConn.Close()

	// Extract authenticated user from permissions
	userID := ""
	username := sshConn.User()
	if sshConn.Permissions != nil && sshConn.Permissions.Extensions != nil {
		userID = sshConn.Permissions.Extensions["user-id"]
		if u := sshConn.Permissions.Extensions["username"]; u != "" {
			username = u
		}
	}

	slog.Info("SSH connection established", "user", username, "user_id", userID, "remote", sshConn.RemoteAddr())

	// Discard global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels (sessions)
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			slog.Error("Failed to accept channel", "error", err)
			continue
		}

		go s.handleSession(channel, requests, username)
	}
}

// handleSession handles an SSH session (git commands)
func (s *SSHServer) handleSession(channel ssh.Channel, requests <-chan *ssh.Request, username string) {
	defer channel.Close()

	for req := range requests {
		switch req.Type {
		case "shell", "exec":
			// Handle git commands
			cmd := ""
			if len(req.Payload) > 0 {
				cmd = string(req.Payload)
			}

			if cmd == "" {
				cmd = "git-upload-pack" // Default git command
			}

			s.handleGitCommand(channel, cmd, username)
			req.Reply(true, nil)

		case "pty-req":
			req.Reply(true, nil)

		case "window-change":
			req.Reply(true, nil)

		default:
			req.Reply(false, nil)
		}
	}
}

// handleGitCommand executes git commands over SSH
func (s *SSHServer) handleGitCommand(channel ssh.Channel, cmd string, username string) {
	slog.Info("SSH git command", "user", username, "cmd", cmd)

	// Parse git command
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	var execCmd *exec.Cmd

	// Handle git-upload-pack and git-receive-pack
	switch parts[0] {
	case "git-upload-pack":
		if len(parts) < 2 {
			slog.Error("Missing repository argument", "cmd", cmd)
			return
		}
		// Extract repo name from path
		repoName := strings.TrimPrefix(parts[1], "/")
		repoName = strings.TrimSuffix(repoName, ".git")
		if !isSafeRepoName(repoName) {
			slog.Error("Invalid repository name in SSH command", "repoName", repoName)
			fmt.Fprintf(channel, "fatal: invalid repository name\n")
			return
		}
		execCmd = exec.Command("git", "upload-pack", "--stateless-rpc", "--advertise-refs", fmt.Sprintf("/tmp/git-repos/%s", repoName))

	case "git-receive-pack":
		if len(parts) < 2 {
			slog.Error("Missing repository argument", "cmd", cmd)
			return
		}
		repoName := strings.TrimPrefix(parts[1], "/")
		repoName = strings.TrimSuffix(repoName, ".git")
		if !isSafeRepoName(repoName) {
			slog.Error("Invalid repository name in SSH command", "repoName", repoName)
			fmt.Fprintf(channel, "fatal: invalid repository name\n")
			return
		}
		execCmd = exec.Command("git", "receive-pack", "--stateless-rpc", fmt.Sprintf("/tmp/git-repos/%s", repoName))

	default:
		slog.Error("Unknown git command", "cmd", parts[0])
		return
	}

	// Set up command I/O
	execCmd.Stdin = channel
	execCmd.Stdout = channel
	execCmd.Stderr = channel

	// Run the command
	if err := execCmd.Run(); err != nil {
		slog.Error("Git command failed", "cmd", cmd, "error", err)
	}
}

// isSafeRepoName validates that a repository name doesn't contain path traversal or special characters
func isSafeRepoName(name string) bool {
	if name == "" {
		return false
	}
	// Reject path traversal attempts
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	// Reject null bytes
	if strings.ContainsRune(name, 0) {
		return false
	}
	// Only allow alphanumeric, hyphens, underscores, and dots
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

// Stop stops the SSH server
func (s *SSHServer) Stop() {
	close(s.done)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
}

// GetPort returns the listening port
func (s *SSHServer) GetPort() int {
	if s.listener == nil {
		return 0
	}
	addr := s.listener.Addr().(*net.TCPAddr)
	return addr.Port
}
