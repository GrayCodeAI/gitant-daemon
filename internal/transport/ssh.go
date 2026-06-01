package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// SSHServer handles SSH connections for git operations
type SSHServer struct {
	sshConfig *ssh.ServerConfig
	listener  net.Listener
	wg        sync.WaitGroup
	done      chan struct{}
	port      int
}

// SSHConfig holds SSH server configuration
type SSHConfig struct {
	Port         int
	HostKeyPath  string
	AuthorizedKeysPath string
}

// NewSSHServer creates a new SSH server
func NewSSHServer(config SSHConfig) (*SSHServer, error) {
	if config.Port == 0 {
		config.Port = 2222
	}

	sshConfig := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Password auth is disabled - only key-based auth
			return nil, fmt.Errorf("password authentication not supported")
		},
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// TODO: Implement authorized_keys check
			// For now, accept all keys for testing
			slog.Info("SSH public key auth", "user", c.User(), "fingerprint", ssh.FingerprintSHA256(key))
			return &ssh.Permissions{}, nil
		},
	}

	// Generate or load host key
	if err := setupHostKey(config.HostKeyPath, sshConfig); err != nil {
		return nil, fmt.Errorf("setting up host key: %w", err)
	}

	server := &SSHServer{
		sshConfig: sshConfig,
		port:      config.Port,
		done:      make(chan struct{}),
	}

	return server, nil
}

// setupHostKey generates or loads the SSH host key
func setupHostKey(hostKeyPath string, config *ssh.ServerConfig) error {
	if hostKeyPath == "" {
		hostKeyPath = "/tmp/gitant_host_key"
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

	slog.Info("SSH connection established", "user", sshConn.User(), "remote", sshConn.RemoteAddr())

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

		go s.handleSession(channel, requests, sshConn.User())
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
		execCmd = exec.Command("git", "upload-pack", "--stateless-rpc", "--advertise-refs", fmt.Sprintf("/tmp/git-repos/%s", repoName))

	case "git-receive-pack":
		if len(parts) < 2 {
			slog.Error("Missing repository argument", "cmd", cmd)
			return
		}
		repoName := strings.TrimPrefix(parts[1], "/")
		repoName = strings.TrimSuffix(repoName, ".git")
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
