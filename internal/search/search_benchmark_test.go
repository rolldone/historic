package search

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func BenchmarkFind1000Files(b *testing.B) {
	workspace, err := config.Initialize(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-benchmark")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		b.Fatal(err)
	}
	for index := 0; index < 1000; index++ {
		metadata := domain.Frontmatter{ID: "00001", Title: "Benchmark Note", Status: domain.StatusProgress, Created: "2026-09-18"}
		document, err := markdown.NewDocument(metadata, "benchmark content")
		if err != nil {
			b.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, filepath.Base(filepath.Join("files", formatBenchmarkName(index)))), document); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		results, err := Find(workspace, Options{Keyword: "benchmark", ActiveOnly: true})
		if err != nil || len(results) != 1000 {
			b.Fatalf("Find = %d, %v", len(results), err)
		}
	}
}

func formatBenchmarkName(index int) string {
	return time.Unix(int64(index), 0).UTC().Format("150405") + "-" + string(rune('a'+index%26)) + ".md"
}
