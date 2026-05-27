package storage

import (
	"fmt"
	"testing"
)

func BenchmarkRepositoryRegistry_List(b *testing.B) {
	dir := b.TempDir()
	reg, _ := NewRepositoryRegistry(dir, dir)

	// Create test repos
	for i := 0; i < 100; i++ {
		reg.Create(fmt.Sprintf("repo-%d", i), fmt.Sprintf("repo-%d", i), "test", false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.List()
	}
}

func BenchmarkRepositoryRegistry_GetEntry(b *testing.B) {
	dir := b.TempDir()
	reg, _ := NewRepositoryRegistry(dir, dir)
	reg.Create("test-repo", "test-repo", "test", false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.GetEntry("test-repo")
	}
}
