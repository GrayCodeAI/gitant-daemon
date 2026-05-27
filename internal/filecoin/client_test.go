package filecoin

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestClient_ProposeDeal(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		MinDealSize:  1024,
		MaxDealSize:  1024 * 1024 * 1024,
		DealDuration: 518400, // 6 months in epochs
		Miners:       []string{"f01234"},
	}
	client := NewClient(cfg, logger)

	deal, err := client.ProposeDeal(context.Background(), "did:gitlawb:repo", "bafyQm...", 10*1024)
	if err != nil {
		t.Fatalf("ProposeDeal failed: %v", err)
	}

	if deal.ID == "" {
		t.Error("Expected non-empty deal ID")
	}
	if deal.Status != DealProposed {
		t.Errorf("Expected status 'proposed', got '%s'", deal.Status)
	}
}

func TestClient_ProposeDeal_TooSmall(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		MinDealSize: 1024,
		MaxDealSize: 1024 * 1024 * 1024,
		Miners:      []string{"f01234"},
	}
	client := NewClient(cfg, logger)

	_, err := client.ProposeDeal(context.Background(), "did:gitlawb:repo", "bafyQm...", 100)
	if err == nil {
		t.Error("Expected error for too small deal")
	}
}

func TestClient_ProposeDeal_TooLarge(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		MinDealSize: 1024,
		MaxDealSize: 1024 * 1024,
		Miners:      []string{"f01234"},
	}
	client := NewClient(cfg, logger)

	_, err := client.ProposeDeal(context.Background(), "did:gitlawb:repo", "bafyQm...", 1024*1024*1024)
	if err == nil {
		t.Error("Expected error for too large deal")
	}
}

func TestClient_GetDeal(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		MinDealSize: 1024,
		MaxDealSize: 1024 * 1024 * 1024,
		Miners:      []string{"f01234"},
	}
	client := NewClient(cfg, logger)

	deal, _ := client.ProposeDeal(context.Background(), "did:gitlawb:repo", "bafyQm...", 10*1024)

	found, err := client.GetDeal(context.Background(), deal.ID)
	if err != nil {
		t.Fatalf("GetDeal failed: %v", err)
	}
	if found.ID != deal.ID {
		t.Errorf("Expected deal ID '%s', got '%s'", deal.ID, found.ID)
	}
}

func TestClient_GetDeal_NotFound(t *testing.T) {
	logger := slog.Default()
	cfg := Config{Miners: []string{"f01234"}}
	client := NewClient(cfg, logger)

	_, err := client.GetDeal(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent deal")
	}
}

func TestClient_ListDeals(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		MinDealSize: 1024,
		MaxDealSize: 1024 * 1024 * 1024,
		Miners:      []string{"f01234"},
	}
	client := NewClient(cfg, logger)

	client.ProposeDeal(context.Background(), "did:gitlawb:repo1", "bafyQm1...", 10*1024)
	client.ProposeDeal(context.Background(), "did:gitlawb:repo1", "bafyQm2...", 20*1024)
	client.ProposeDeal(context.Background(), "did:gitlawb:repo2", "bafyQm3...", 30*1024)

	deals := client.ListDeals(context.Background(), "did:gitlawb:repo1")
	if len(deals) != 2 {
		t.Errorf("Expected 2 deals for repo1, got %d", len(deals))
	}
}

func TestClient_ArchiveRepo(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		MinDealSize:  1024,
		MaxDealSize:  1024 * 1024 * 1024,
		Miners:       []string{"f01234"},
		AutoArchive:  true,
		ArchiveAfter: 30 * 24 * time.Hour,
	}
	client := NewClient(cfg, logger)

	// Repo too young
	_, err := client.ArchiveRepo(context.Background(), "did:gitlawb:repo", "bafyQm...", 10*1024, 10*24*time.Hour)
	if err == nil {
		t.Error("Expected error for too young repo")
	}

	// Repo old enough
	deal, err := client.ArchiveRepo(context.Background(), "did:gitlawb:repo", "bafyQm...", 10*1024, 60*24*time.Hour)
	if err != nil {
		t.Fatalf("ArchiveRepo failed: %v", err)
	}
	if deal == nil {
		t.Error("Expected non-nil deal")
	}
}

func TestClient_GetStats(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		MinDealSize: 1024,
		MaxDealSize: 1024 * 1024 * 1024,
		Miners:      []string{"f01234"},
	}
	client := NewClient(cfg, logger)

	client.ProposeDeal(context.Background(), "did:gitlawb:repo", "bafyQm...", 10*1024)

	stats := client.GetStats()
	if stats["total_deals"] != 1 {
		t.Errorf("Expected 1 total deal, got %v", stats["total_deals"])
	}
}
