package crdt

import (
	"fmt"
	"testing"
)

func BenchmarkIssueStore_Create(b *testing.B) {
	dir := b.TempDir()
	store := NewIssueStore(dir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Create("test-repo", fmt.Sprintf("issue-%d", i), "author", fmt.Sprintf("Issue %d", i), "Test body")
	}
}

func BenchmarkIssueStore_List(b *testing.B) {
	dir := b.TempDir()
	store := NewIssueStore(dir)

	// Create test issues
	for i := 0; i < 100; i++ {
		store.Create("test-repo", fmt.Sprintf("issue-%d", i), "author", fmt.Sprintf("Issue %d", i), "Test body")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.List("test-repo")
	}
}

func BenchmarkIssueStore_Get(b *testing.B) {
	dir := b.TempDir()
	store := NewIssueStore(dir)
	store.Create("test-repo", "test-issue", "author", "Test Issue", "Test body")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Get("test-repo", "test-issue")
	}
}

func BenchmarkPullRequestStore_Create(b *testing.B) {
	dir := b.TempDir()
	store := NewPullRequestStore(dir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Create("test-repo", fmt.Sprintf("pr-%d", i), "author", fmt.Sprintf("PR %d", i), "Test body", "feature", "main")
	}
}

func BenchmarkPullRequestStore_List(b *testing.B) {
	dir := b.TempDir()
	store := NewPullRequestStore(dir)

	// Create test PRs
	for i := 0; i < 100; i++ {
		store.Create("test-repo", fmt.Sprintf("pr-%d", i), "author", fmt.Sprintf("PR %d", i), "Test body", "feature", "main")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.List("test-repo")
	}
}
