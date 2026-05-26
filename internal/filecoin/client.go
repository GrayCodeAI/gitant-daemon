package filecoin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// DealStatus represents the status of a Filecoin deal
type DealStatus string

const (
	DealProposed  DealStatus = "proposed"
	DealPublished DealStatus = "published"
	DealActive    DealStatus = "active"
	DealExpired   DealStatus = "expired"
	DealSlashed   DealStatus = "slashed"
	DealFailed    DealStatus = "failed"
)

// Deal represents a Filecoin storage deal
type Deal struct {
	ID          string     `json:"id"`
	RepoDID     string     `json:"repo_did"`
	CID         string     `json:"cid"`
	Size        uint64     `json:"size"`
	Provider    string     `json:"provider"`
	Price       string     `json:"price"`
	Duration    uint64     `json:"duration"` // epochs
	Status      DealStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Config for Filecoin client
type Config struct {
	APIEndpoint  string
	AuthToken    string
	MinDealSize  uint64
	MaxDealSize  uint64
	DealDuration uint64 // epochs (1 epoch = 30s)
	Miners       []string
	AutoArchive  bool
	ArchiveAfter time.Duration
}

// Client manages Filecoin storage deals
type Client struct {
	cfg     Config
	logger  *slog.Logger
	mu      sync.RWMutex
	deals   map[string]*Deal
	pending []string
}

// NewClient creates a new Filecoin client
func NewClient(cfg Config, logger *slog.Logger) *Client {
	return &Client{
		cfg:    cfg,
		logger: logger,
		deals:  make(map[string]*Deal),
	}
}

// ProposeDeal proposes a new storage deal
func (c *Client) ProposeDeal(ctx context.Context, repoDID, cid string, size uint64) (*Deal, error) {
	if size < c.cfg.MinDealSize {
		return nil, fmt.Errorf("size %d below minimum %d", size, c.cfg.MinDealSize)
	}
	if size > c.cfg.MaxDealSize {
		return nil, fmt.Errorf("size %d above maximum %d", size, c.cfg.MaxDealSize)
	}

	if len(c.cfg.Miners) == 0 {
		return nil, fmt.Errorf("no miners configured")
	}

	deal := &Deal{
		ID:        fmt.Sprintf("deal-%d", time.Now().UnixNano()),
		RepoDID:   repoDID,
		CID:       cid,
		Size:      size,
		Provider:  c.cfg.Miners[0], // TODO: miner selection
		Status:    DealProposed,
		CreatedAt: time.Now(),
		Duration:  c.cfg.DealDuration,
	}

	c.mu.Lock()
	c.deals[deal.ID] = deal
	c.pending = append(c.pending, deal.ID)
	c.mu.Unlock()

	c.logger.Info("deal proposed",
		"deal_id", deal.ID,
		"repo", repoDID,
		"cid", cid,
		"size", size,
		"provider", deal.Provider,
	)

	return deal, nil
}

// GetDeal returns a deal by ID
func (c *Client) GetDeal(ctx context.Context, dealID string) (*Deal, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	deal, exists := c.deals[dealID]
	if !exists {
		return nil, fmt.Errorf("deal %s not found", dealID)
	}
	return deal, nil
}

// ListDeals returns all deals for a repo
func (c *Client) ListDeals(ctx context.Context, repoDID string) []*Deal {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var deals []*Deal
	for _, deal := range c.deals {
		if deal.RepoDID == repoDID {
			deals = append(deals, deal)
		}
	}
	return deals
}

// CheckDealStatus checks the current status of a deal
func (c *Client) CheckDealStatus(ctx context.Context, dealID string) (DealStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	deal, exists := c.deals[dealID]
	if !exists {
		return "", fmt.Errorf("deal %s not found", dealID)
	}

	// TODO: Query Lotus API for actual deal status
	return deal.Status, nil
}

// ArchiveRepo archives a repo to Filecoin if it meets criteria
func (c *Client) ArchiveRepo(ctx context.Context, repoDID, cid string, size uint64, age time.Duration) (*Deal, error) {
	if !c.cfg.AutoArchive {
		return nil, fmt.Errorf("auto-archive disabled")
	}
	if age < c.cfg.ArchiveAfter {
		return nil, fmt.Errorf("repo too young for archival: %v < %v", age, c.cfg.ArchiveAfter)
	}

	return c.ProposeDeal(ctx, repoDID, cid, size)
}

// GetPendingDeals returns deals awaiting activation
func (c *Client) GetPendingDeals() []*Deal {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var pending []*Deal
	for _, id := range c.pending {
		if deal, ok := c.deals[id]; ok && deal.Status == DealProposed {
			pending = append(pending, deal)
		}
	}
	return pending
}

// GetStats returns storage statistics
func (c *Client) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalSize uint64
	activeCount := 0
	for _, deal := range c.deals {
		totalSize += deal.Size
		if deal.Status == DealActive {
			activeCount++
		}
	}

	return map[string]interface{}{
		"total_deals":   len(c.deals),
		"active_deals":  activeCount,
		"pending_deals": len(c.pending),
		"total_size":    totalSize,
	}
}
