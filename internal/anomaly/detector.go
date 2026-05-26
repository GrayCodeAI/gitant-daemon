package anomaly

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// EventType represents a tracked event type
type EventType string

const (
	EventPush      EventType = "push"
	EventPRCreate  EventType = "pr_create"
	EventPRMerge   EventType = "pr_merge"
	EventIssue     EventType = "issue"
	EventComment   EventType = "comment"
	EventAPIKey    EventType = "api_key"
	EventUCANIssue EventType = "ucan_issue"
	EventAuthFail  EventType = "auth_fail"
)

// Event represents a tracked event
type Event struct {
	Type      EventType
	DID       string
	Timestamp time.Time
	IP        string
	Success   bool
	Metadata  map[string]interface{}
}

// Baseline represents normal behavior for a DID
type Baseline struct {
	DID              string    `json:"did"`
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
	TotalEvents      int       `json:"total_events"`
	EventCounts      map[EventType]int `json:"event_counts"`
	AvgEventsPerHour float64   `json:"avg_events_per_hour"`
	UniqueIPs        int       `json:"unique_ips"`
	AuthFailRate     float64   `json:"auth_fail_rate"`
	TrustScore       float64   `json:"trust_score"`
}

// Alert represents an anomaly alert
type Alert struct {
	DID       string    `json:"did"`
	EventType EventType `json:"event_type"`
	Reason    string    `json:"reason"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
	Details   map[string]interface{} `json:"details"`
}

// Detector detects anomalies in agent behavior
type Detector struct {
	mu           sync.RWMutex
	logger       *slog.Logger
	baselines    map[string]*Baseline
	recentEvents map[string][]Event
	alerts       []Alert
	windowSize   time.Duration
	thresholds   Thresholds
}

// Thresholds for anomaly detection
type Thresholds struct {
	MaxEventsPerHour   float64
	MaxAuthFailRate    float64
	MaxUniqueIPsPerDay int
	BurstThreshold     int
	BurstWindow        time.Duration
}

// DefaultThresholds returns sensible defaults
func DefaultThresholds() Thresholds {
	return Thresholds{
		MaxEventsPerHour:   100,
		MaxAuthFailRate:    0.3,
		MaxUniqueIPsPerDay: 10,
		BurstThreshold:     50,
		BurstWindow:        1 * time.Minute,
	}
}

// NewDetector creates a new anomaly detector
func NewDetector(logger *slog.Logger, windowSize time.Duration, thresholds Thresholds) *Detector {
	return &Detector{
		logger:       logger,
		baselines:    make(map[string]*Baseline),
		recentEvents: make(map[string][]Event),
		windowSize:   windowSize,
		thresholds:   thresholds,
	}
}

// Track records an event and checks for anomalies
func (d *Detector) Track(ctx context.Context, event Event) *Alert {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Update baseline
	baseline := d.getOrCreateBaseline(event.DID)
	baseline.LastSeen = event.Timestamp
	baseline.TotalEvents++
	baseline.EventCounts[event.Type]++

	// Track recent events
	d.recentEvents[event.DID] = append(d.recentEvents[event.DID], event)
	d.pruneOldEvents(event.DID)

	// Update stats
	d.updateBaselineStats(baseline, event.DID)

	// Check for anomalies
	return d.checkAnomalies(event, baseline)
}

// GetBaseline returns the baseline for a DID
func (d *Detector) GetBaseline(did string) *Baseline {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.baselines[did]
}

// GetAlerts returns all alerts
func (d *Detector) GetAlerts() []Alert {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.alerts
}

// GetAlertsForDID returns alerts for a specific DID
func (d *Detector) GetAlertsForDID(did string) []Alert {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var alerts []Alert
	for _, alert := range d.alerts {
		if alert.DID == did {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

func (d *Detector) getOrCreateBaseline(did string) *Baseline {
	baseline, exists := d.baselines[did]
	if !exists {
		baseline = &Baseline{
			DID:         did,
			FirstSeen:   time.Now(),
			EventCounts: make(map[EventType]int),
		}
		d.baselines[did] = baseline
	}
	return baseline
}

func (d *Detector) pruneOldEvents(did string) {
	cutoff := time.Now().Add(-d.windowSize)
	events := d.recentEvents[did]
	pruned := make([]Event, 0, len(events))
	for _, e := range events {
		if e.Timestamp.After(cutoff) {
			pruned = append(pruned, e)
		}
	}
	d.recentEvents[did] = pruned
}

func (d *Detector) updateBaselineStats(baseline *Baseline, did string) {
	events := d.recentEvents[did]
	if len(events) == 0 {
		return
	}

	// Calculate events per hour
	if !baseline.FirstSeen.IsZero() {
		hours := time.Since(baseline.FirstSeen).Hours()
		if hours > 0 {
			baseline.AvgEventsPerHour = float64(baseline.TotalEvents) / hours
		}
	}

	// Count unique IPs
	ips := make(map[string]bool)
	authFails := 0
	for _, e := range events {
		if e.IP != "" {
			ips[e.IP] = true
		}
		if e.Type == EventAuthFail {
			authFails++
		}
	}
	baseline.UniqueIPs = len(ips)

	if len(events) > 0 {
		baseline.AuthFailRate = float64(authFails) / float64(len(events))
	}
}

func (d *Detector) checkAnomalies(event Event, baseline *Baseline) *Alert {
	// Check burst
	events := d.recentEvents[event.DID]
	burstCount := 0
	burstCutoff := time.Now().Add(-d.thresholds.BurstWindow)
	for _, e := range events {
		if e.Timestamp.After(burstCutoff) {
			burstCount++
		}
	}
	if burstCount > d.thresholds.BurstThreshold {
		alert := Alert{
			DID:       event.DID,
			EventType: event.Type,
			Reason:    "burst_detected",
			Severity:  "high",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"burst_count": burstCount,
				"threshold":   d.thresholds.BurstThreshold,
			},
		}
		d.alerts = append(d.alerts, alert)
		d.logger.Warn("anomaly detected", "did", event.DID, "reason", "burst", "count", burstCount)
		return &alert
	}

	// Check auth fail rate
	if baseline.AuthFailRate > d.thresholds.MaxAuthFailRate {
		alert := Alert{
			DID:       event.DID,
			EventType: EventAuthFail,
			Reason:    "high_auth_fail_rate",
			Severity:  "medium",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"fail_rate": baseline.AuthFailRate,
				"threshold": d.thresholds.MaxAuthFailRate,
			},
		}
		d.alerts = append(d.alerts, alert)
		d.logger.Warn("anomaly detected", "did", event.DID, "reason", "auth_fail_rate", "rate", baseline.AuthFailRate)
		return &alert
	}

	// Check unique IPs
	if baseline.UniqueIPs > d.thresholds.MaxUniqueIPsPerDay {
		alert := Alert{
			DID:       event.DID,
			EventType: event.Type,
			Reason:    "too_many_ips",
			Severity:  "low",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"unique_ips": baseline.UniqueIPs,
				"threshold":  d.thresholds.MaxUniqueIPsPerDay,
			},
		}
		d.alerts = append(d.alerts, alert)
		d.logger.Warn("anomaly detected", "did", event.DID, "reason", "too_many_ips", "count", baseline.UniqueIPs)
		return &alert
	}

	return nil
}

// ShouldRevoke checks if a DID should have its UCANs revoked
func (d *Detector) ShouldRevoke(did string) (bool, string) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	baseline, exists := d.baselines[did]
	if !exists {
		return false, ""
	}

	// High severity = revoke
	for _, alert := range d.alerts {
		if alert.DID == did && alert.Severity == "high" {
			return true, alert.Reason
		}
	}

	// Multiple medium alerts = revoke
	mediumCount := 0
	for _, alert := range d.alerts {
		if alert.DID == did && alert.Severity == "medium" {
			mediumCount++
		}
	}
	if mediumCount >= 3 {
		return true, "multiple_medium_alerts"
	}

	// Very high auth fail rate
	if baseline.AuthFailRate > 0.5 {
		return true, "critical_auth_fail_rate"
	}

	return false, ""
}
