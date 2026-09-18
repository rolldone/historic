package search

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

func BenchmarkFind1000Files(b *testing.B) {
	benchmarkFindFiles(b, 1000)
}

func BenchmarkFind10000Files(b *testing.B) {
	benchmarkFindFiles(b, 10000)
}

func BenchmarkFind100000Files(b *testing.B) {
	benchmarkFindFiles(b, 100000)
}

func benchmarkFindFiles(b *testing.B, total int) {
	b.Helper()
	workspace, err := config.Initialize(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-benchmark")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		b.Fatal(err)
	}
	metadata := domain.Frontmatter{ID: "00001", Title: "Benchmark Note", Status: domain.StatusProgress, Created: "2026-09-18"}
	for index := 0; index < total; index++ {
		document, err := markdown.NewDocument(metadata, "benchmark content")
		if err != nil {
			b.Fatal(err)
		}
		path := filepath.Join(topic, fmt.Sprintf("%05d-benchmark.md", index))
		if err := markdown.WriteFile(path, document); err != nil {
			b.Fatal(err)
		}
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		results, err := Find(workspace, Options{Keyword: "benchmark", ActiveOnly: true})
		if err != nil || len(results) != total {
			b.Fatalf("Find = %d, want %d: %v", len(results), total, err)
		}
	}
}
