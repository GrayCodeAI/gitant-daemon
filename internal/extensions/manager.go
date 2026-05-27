package extensions

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"plugin"
	"sync"
)

// Plugin represents a loaded plugin
type Plugin struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Path        string `json:"path"`
	Enabled     bool   `json:"enabled"`
}

// PluginAPI is the interface plugins must implement
type PluginAPI interface {
	// Init initializes the plugin
	Init(config map[string]interface{}) error

	// OnRepoCreated is called when a repo is created
	OnRepoCreated(repoID string) error

	// OnIssueCreated is called when an issue is created
	OnIssueCreated(repoID, issueID string) error

	// OnPRMerged is called when a PR is merged
	OnPRMerged(repoID, prID string) error

	// OnPush is called when code is pushed
	OnPush(repoID, ref string) error

	// Shutdown cleans up the plugin
	Shutdown() error
}

// Manager manages plugins
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]*Plugin
	apis    map[string]PluginAPI
	dir     string
}

// NewManager creates a new plugin manager
func NewManager(dir string) *Manager {
	return &Manager{
		plugins: make(map[string]*Plugin),
		apis:    make(map[string]PluginAPI),
		dir:     dir,
	}
}

// Load loads all plugins from the plugins directory
func (m *Manager) Load() error {
	if err := os.MkdirAll(m.dir, 0755); err != nil {
		return fmt.Errorf("creating plugins dir: %w", err)
	}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return fmt.Errorf("reading plugins dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".so" {
			continue
		}

		path := filepath.Join(m.dir, entry.Name())
		if err := m.loadPlugin(path); err != nil {
			slog.Warn("failed to load plugin", "path", path, "error", err)
		}
	}

	return nil
}

func (m *Manager) loadPlugin(path string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("opening plugin: %w", err)
	}

	symAPI, err := p.Lookup("API")
	if err != nil {
		return fmt.Errorf("plugin does not export API symbol: %w", err)
	}

	api, ok := symAPI.(PluginAPI)
	if !ok {
		return fmt.Errorf("plugin API does not implement PluginAPI interface")
	}

	name := filepath.Base(path)
	pluginInfo := &Plugin{
		Name:    name,
		Path:    path,
		Enabled: true,
	}

	m.mu.Lock()
	m.plugins[name] = pluginInfo
	m.apis[name] = api
	m.mu.Unlock()

	slog.Info("loaded plugin", "name", name, "path", path)
	return nil
}

// Enable enables a plugin
func (m *Manager) Enable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	p.Enabled = true
	return nil
}

// Disable disables a plugin
func (m *Manager) Disable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	p.Enabled = false
	return nil
}

// List returns all plugins
func (m *Manager) List() []*Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// Get returns a plugin by name
func (m *Manager) Get(name string) (*Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}
	return p, nil
}

// Install installs a plugin from a file
func (m *Manager) Install(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading plugin: %w", err)
	}

	dest := filepath.Join(m.dir, filepath.Base(path))
	if err := os.WriteFile(dest, data, 0755); err != nil {
		return fmt.Errorf("writing plugin: %w", err)
	}

	return m.loadPlugin(dest)
}

// Uninstall removes a plugin
func (m *Manager) Uninstall(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	if api, ok := m.apis[name]; ok {
		if err := api.Shutdown(); err != nil {
			slog.Warn("plugin shutdown error", "name", name, "error", err)
		}
	}

	if err := os.Remove(p.Path); err != nil {
		return fmt.Errorf("removing plugin: %w", err)
	}

	delete(m.plugins, name)
	delete(m.apis, name)
	return nil
}

// NotifyRepoCreated notifies all plugins of repo creation
func (m *Manager) NotifyRepoCreated(repoID string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, api := range m.apis {
		p := m.plugins[name]
		if !p.Enabled {
			continue
		}
		if err := api.OnRepoCreated(repoID); err != nil {
			slog.Warn("plugin error", "name", name, "event", "repo_created", "error", err)
		}
	}
}

// NotifyIssueCreated notifies all plugins of issue creation
func (m *Manager) NotifyIssueCreated(repoID, issueID string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, api := range m.apis {
		p := m.plugins[name]
		if !p.Enabled {
			continue
		}
		if err := api.OnIssueCreated(repoID, issueID); err != nil {
			slog.Warn("plugin error", "name", name, "event", "issue_created", "error", err)
		}
	}
}

// NotifyPRMerged notifies all plugins of PR merge
func (m *Manager) NotifyPRMerged(repoID, prID string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, api := range m.apis {
		p := m.plugins[name]
		if !p.Enabled {
			continue
		}
		if err := api.OnPRMerged(repoID, prID); err != nil {
			slog.Warn("plugin error", "name", name, "event", "pr_merged", "error", err)
		}
	}
}

// NotifyPush notifies all plugins of push
func (m *Manager) NotifyPush(repoID, ref string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, api := range m.apis {
		p := m.plugins[name]
		if !p.Enabled {
			continue
		}
		if err := api.OnPush(repoID, ref); err != nil {
			slog.Warn("plugin error", "name", name, "event", "push", "error", err)
		}
	}
}

// Shutdown shuts down all plugins
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, api := range m.apis {
		if err := api.Shutdown(); err != nil {
			slog.Warn("plugin shutdown error", "name", name, "error", err)
		}
	}
}

// Save saves plugin configuration
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := filepath.Join(m.dir, "plugins.json")
	data, err := json.MarshalIndent(m.plugins, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Load loads plugin configuration
func (m *Manager) LoadConfig() error {
	path := filepath.Join(m.dir, "plugins.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &m.plugins)
}
