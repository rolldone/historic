package repository

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/domain"
)

func TestAddEntryCreatesMarkdownAndUpdatesMeta(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Admin Dashboard", "00014")
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.AddEntry(topic.ID, "prd", false)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	if entry.Filename != "prd.md" {
		t.Fatalf("filename = %q, want prd.md", entry.Filename)
	}
	if _, err := os.Stat(filepath.Join(topic.Path, "prd.md")); err != nil {
		t.Fatal(err)
	}
	meta, err := os.ReadFile(filepath.Join(topic.Path, "_meta.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(meta), "./prd.md") {
		t.Fatalf("meta missing entry link: %s", meta)
	}
	if strings.Contains(string(meta), `\n`) {
		t.Fatalf("meta contains literal escaped newline: %q", meta)
	}
}

func TestNormalizeEntryName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "sentence", input: "Remove stale Cost Dashboard breakdowns", want: "remove-stale-cost-dashboard-breakdowns.md"},
		{name: "trim and collapse spaces", input: "  Multiple   spaces  ", want: "multiple-spaces.md"},
		{name: "existing slug", input: "Already-slugged-name", want: "already-slugged-name.md"},
		{name: "underscore", input: "name_with_separator", want: "name-with-separator.md"},
		{name: "repeated separators", input: "name__--with---separators.md.md", want: "name-with-separators.md"},
		{name: "uppercase extension", input: "FILE.MD", want: "file.md"},
		{name: "work order prefix", input: "wos/Review Login Flow", want: "wos/review-login-flow.md"},
		{name: "explicit work order number", input: "wos/01-Review Login Flow", want: "wos/01-review-login-flow.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeEntryName(tt.input)
			if err != nil {
				t.Fatalf("normalizeEntryName(%q): %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeEntryName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeEntryNameRejectsUnsafeNames(t *testing.T) {
	for _, name := range []string{"", "///", "../escape", `..\\escape`, "/tmp/absolute", `C:\\tmp\\absolute`, ".hidden", "wos/"} {
		if _, err := normalizeEntryName(name); !errors.Is(err, domain.ErrConflict) {
			t.Errorf("normalizeEntryName(%q) error = %v, want ErrConflict", name, err)
		}
	}
}

func TestAddEntryNumbersWorkOrder(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Topic", "00001")
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.AddEntry(topic.ID, "wos/scaffold", false)
	if err != nil {
		t.Fatalf("first AddEntry: %v", err)
	}
	second, err := store.AddEntry(topic.ID, "wos/review", false)
	if err != nil {
		t.Fatalf("second AddEntry: %v", err)
	}
	if first.Filename != "wos/01-scaffold.md" || second.Filename != "wos/02-review.md" {
		t.Fatalf("files = %q, %q", first.Filename, second.Filename)
	}
	meta, err := os.ReadFile(filepath.Join(topic.Path, "_meta.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(meta)
	if !strings.Contains(content, "\n- [01-scaffold.md](./wos/01-scaffold.md)\n") || !strings.Contains(content, "\n- [02-review.md](./wos/02-review.md)\n") {
		t.Fatalf("work orders not formatted as Markdown list: %q", content)
	}
	if strings.Contains(content, `\n`) {
		t.Fatalf("meta contains literal escaped newline: %q", content)
	}
}

func TestAddEntryCanonicalCollisionAndForce(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Topic", "00001")
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.AddEntry(topic.ID, "Remove stale Cost Dashboard breakdowns", false)
	if err != nil {
		t.Fatal(err)
	}
	if first.Filename != "remove-stale-cost-dashboard-breakdowns.md" {
		t.Fatalf("filename = %q", first.Filename)
	}
	if _, err := store.AddEntry(topic.ID, "remove_stale cost dashboard breakdowns.md", false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("canonical collision error = %v, want ErrConflict", err)
	}
	if _, err := store.AddEntry(topic.ID, "remove_stale cost dashboard breakdowns.md", true); err != nil {
		t.Fatalf("forced canonical overwrite: %v", err)
	}
}

func TestAddEntryRejectsTraversalAndOverwrite(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Topic", "00001")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../escape", "/tmp/absolute"} {
		if _, err := store.AddEntry(topic.ID, name, false); !errors.Is(err, domain.ErrConflict) {
			t.Errorf("AddEntry(%q) error = %v, want ErrConflict", name, err)
		}
	}
	if _, err := store.AddEntry(topic.ID, "note", false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddEntry(topic.ID, "note", false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("overwrite error = %v, want ErrConflict", err)
	}
	if _, err := store.AddEntry(topic.ID, "note", true); err != nil {
		t.Fatalf("forced overwrite: %v", err)
	}
}
