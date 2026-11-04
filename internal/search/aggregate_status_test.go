package search

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

func writeAggregateDocument(t *testing.T, path string, metadata domain.Frontmatter, body string) {
	t.Helper()
	document, err := markdown.NewDocument(metadata, body)
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(path, document); err != nil {
		t.Fatal(err)
	}
}

func TestFindStatusMatchesFilesAndAggregatedTopics(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-aggregate")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{ID: "00001", Title: "Aggregate", Created: "2026-09-18"}); err != nil {
		t.Fatal(err)
	}
	writeAggregateDocument(t, filepath.Join(topic, "done.md"), domain.Frontmatter{ID: "00001", Title: "Done", Status: domain.StatusComplete, Created: "2026-09-18"}, "aggregate work")
	writeAggregateDocument(t, filepath.Join(topic, "moved.md"), domain.Frontmatter{ID: "00001", Title: "Moved", Status: domain.StatusForwarded, Created: "2026-09-18"}, "aggregate work")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	results, err := Find(workspace, Options{Keyword: "aggregate", Status: domain.StatusComplete})
	if err != nil {
		t.Fatal(err)
	}
	sawTopic, sawFile := false, false
	for _, result := range results {
		switch result.Type {
		case "topic":
			if result.Status != "complete" {
				t.Fatalf("topic status = %q, want complete", result.Status)
			}
			sawTopic = true
		case "historic_file":
			if result.Status != "complete" {
				t.Fatalf("file status = %q, want complete", result.Status)
			}
			sawFile = true
		}
	}
	if !sawTopic || !sawFile {
		t.Fatalf("status filter results missing matches: topic=%v file=%v (%+v)", sawTopic, sawFile, results)
	}

	excluded, err := Find(workspace, Options{Keyword: "aggregate", StatusNot: []domain.Status{domain.StatusComplete}})
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range excluded {
		if result.Type == "topic" && result.Status == "complete" {
			t.Fatalf("complete topic leaked into status-not filter: %+v", result)
		}
		if result.Type == "historic_file" && result.Status == "complete" {
			t.Fatalf("complete file leaked into status-not filter: %+v", result)
		}
	}
}

func TestFindTopicContextExposesProgressPercent(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(workspace.Histories, "00001-partial")
	empty := filepath.Join(workspace.Histories, "00002-empty")
	for _, topic := range []string{partial, empty} {
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(partial, markdown.MetaFilename), markdown.TopicMetadata{ID: "00001", Title: "Partial", Created: "2026-09-18"}); err != nil {
		t.Fatal(err)
	}
	writeAggregateDocument(t, filepath.Join(partial, "done.md"), domain.Frontmatter{ID: "00001", Title: "Done", Status: domain.StatusComplete, Created: "2026-09-18"}, "partial token")
	writeAggregateDocument(t, filepath.Join(partial, "open.md"), domain.Frontmatter{ID: "00001", Title: "Open", Status: domain.StatusProgress, Created: "2026-09-18"}, "partial token")
	if err := markdown.WriteTopicMetadata(filepath.Join(empty, markdown.MetaFilename), markdown.TopicMetadata{ID: "00002", Title: "Empty", Created: "2026-09-18"}); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	results, err := Find(workspace, Options{Keyword: "partial"})
	if err != nil {
		t.Fatal(err)
	}
	var topicResult *Result
	for index := range results {
		if results[index].Type == "topic" && results[index].TopicID == "00001" {
			topicResult = &results[index]
		}
	}
	if topicResult == nil || topicResult.Topic.ProgressPercent == nil || *topicResult.Topic.ProgressPercent != 50 {
		t.Fatalf("topic context progress = %+v", topicResult)
	}

	recent, err := RecentTopics(workspace, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var emptyResult *Result
	for index := range recent {
		if recent[index].TopicID == "00002" {
			emptyResult = &recent[index]
		}
	}
	if emptyResult == nil || emptyResult.Topic.ProgressPercent != nil {
		t.Fatalf("empty recent topic progress = %+v", emptyResult)
	}
}

func TestFindRejectsSyntheticActiveStatus(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Find(workspace, Options{Keyword: "anything", Status: domain.Status("active")}); err == nil {
		t.Fatal("--status active must be rejected as an unknown status")
	}
	if _, err := Find(workspace, Options{Keyword: "anything", StatusNot: []domain.Status{domain.Status("active")}}); err == nil {
		t.Fatal("--status-not active must be rejected as an unknown status")
	}
}
