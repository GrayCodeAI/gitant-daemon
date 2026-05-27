package anomaly

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestDetector_Track(t *testing.T) {
	logger := slog.Default()
	detector := NewDetector(logger, 1*time.Hour, DefaultThresholds())

	event := Event{
		Type:      EventPush,
		DID:       "did:key:test",
		Timestamp: time.Now(),
		IP:        "127.0.0.1",
		Success:   true,
	}

	alert := detector.Track(context.TODO(), event)
	if alert != nil {
		t.Errorf("Expected no alert, got: %v", alert)
	}

	baseline := detector.GetBaseline("did:key:test")
	if baseline == nil {
		t.Fatal("Expected baseline to be created")
	}
	if baseline.TotalEvents != 1 {
		t.Errorf("Expected 1 event, got %d", baseline.TotalEvents)
	}
}

func TestDetector_Track_Burst(t *testing.T) {
	logger := slog.Default()
	thresholds := DefaultThresholds()
	thresholds.BurstThreshold = 5
	detector := NewDetector(logger, 1*time.Hour, thresholds)

	var lastAlert *Alert
	for i := 0; i < 10; i++ {
		event := Event{
			Type:      EventPush,
			DID:       "did:key:test",
			Timestamp: time.Now(),
			IP:        "127.0.0.1",
			Success:   true,
		}
		lastAlert = detector.Track(context.TODO(), event)
	}

	if lastAlert == nil {
		t.Error("Expected burst alert")
	}
	if lastAlert != nil && lastAlert.Reason != "burst_detected" {
		t.Errorf("Expected reason 'burst_detected', got '%s'", lastAlert.Reason)
	}
}

func TestDetector_Track_AuthFailRate(t *testing.T) {
	logger := slog.Default()
	thresholds := DefaultThresholds()
	thresholds.MaxAuthFailRate = 0.5
	detector := NewDetector(logger, 1*time.Hour, thresholds)

	// Generate events with high auth fail rate
	for i := 0; i < 10; i++ {
		event := Event{
			Type:      EventAuthFail,
			DID:       "did:key:test",
			Timestamp: time.Now(),
			IP:        "127.0.0.1",
			Success:   false,
		}
		detector.Track(context.TODO(), event)
	}

	baseline := detector.GetBaseline("did:key:test")
	if baseline.AuthFailRate <= 0.5 {
		t.Errorf("Expected high auth fail rate, got %f", baseline.AuthFailRate)
	}
}

func TestDetector_ShouldRevoke(t *testing.T) {
	logger := slog.Default()
	thresholds := DefaultThresholds()
	thresholds.BurstThreshold = 3
	detector := NewDetector(logger, 1*time.Hour, thresholds)

	// Generate burst events
	for i := 0; i < 10; i++ {
		event := Event{
			Type:      EventPush,
			DID:       "did:key:test",
			Timestamp: time.Now(),
			IP:        "127.0.0.1",
		}
		detector.Track(context.TODO(), event)
	}

	shouldRevoke, reason := detector.ShouldRevoke("did:key:test")
	if !shouldRevoke {
		t.Error("Expected revocation recommendation")
	}
	if reason == "" {
		t.Error("Expected non-empty reason")
	}
}

func TestDetector_GetAlertsForDID(t *testing.T) {
	logger := slog.Default()
	thresholds := DefaultThresholds()
	thresholds.BurstThreshold = 3
	detector := NewDetector(logger, 1*time.Hour, thresholds)

	// Generate events for two DIDs
	for i := 0; i < 10; i++ {
		detector.Track(context.TODO(), Event{
			Type:      EventPush,
			DID:       "did:key:user1",
			Timestamp: time.Now(),
			IP:        "127.0.0.1",
		})
	}

	for i := 0; i < 2; i++ {
		detector.Track(context.TODO(), Event{
			Type:      EventPush,
			DID:       "did:key:user2",
			Timestamp: time.Now(),
			IP:        "127.0.0.1",
		})
	}

	alerts1 := detector.GetAlertsForDID("did:key:user1")
	alerts2 := detector.GetAlertsForDID("did:key:user2")

	if len(alerts1) == 0 {
		t.Error("Expected alerts for user1")
	}
	if len(alerts2) != 0 {
		t.Errorf("Expected no alerts for user2, got %d", len(alerts2))
	}
}
