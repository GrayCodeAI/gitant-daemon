package webhooks

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkManager_Dispatch(b *testing.B) {
	mgr := NewManager()

	// Register test webhook
	mgr.Register("test-webhook", "https://example.com/webhook", []EventType{EventPush, EventIssueCreated}, "")

	event := Event{
		Type:      EventPush,
		Repo:      "test-repo",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"ref": "refs/heads/main"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.Dispatch(event)
	}
}

func BenchmarkManager_Register(b *testing.B) {
	mgr := NewManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.Register(fmt.Sprintf("webhook-%d", i), "https://example.com/webhook", []EventType{EventPush}, "")
	}
}

func BenchmarkValidateWebhookURL(b *testing.B) {
	urls := []string{
		"https://example.com/webhook",
		"https://hooks.slack.com/services/xxx",
		"https://discord.com/api/webhooks/xxx",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateWebhookURL(urls[i%len(urls)])
	}
}
