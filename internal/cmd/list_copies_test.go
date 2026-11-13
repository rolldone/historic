package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/display"
	"historic/internal/domain"
)

// displayDividerToken is the rule character used to separate the two mirrored
// copies of one topic. Tests assert on a run of it rather than on the full
// terminal width so the assertion stays independent of COLUMNS.
const displayDividerToken = "─────"

// countListRows counts lines that begin with the topic ID (the row prefix),
// excluding the same digits appearing inside a printed path.
func countListRows(output, id string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, id+" ") {
			count++
		}
	}
	return count
}

// renderNewerCopyNoticeForTest exposes the notice renderer to tests so the
// no-TTY behaviour can be asserted without a real terminal.
func renderNewerCopyNoticeForTest(out io.Writer, region string) string {
	return display.NewerCopyNotice(out, domain.CopyRegionCaption(region), false)
}

// writeMirroredTopicFixture writes the same topic into both storages with
// caller-chosen stamps, so the direction of the freshness notice is explicit.
func writeMirroredTopicFixture(t *testing.T, root, id, title, openStamp, closedStamp string) {
	t.Helper()
	for _, base := range []string{filepath.Join(root, ".historic"), filepath.Join(root, ".historic", ".database")} {
		topic := filepath.Join(base, id+"-mirrored")
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		content := "id: " + id + "\ntitle: " + title + "\ncreated: 2026-10-01\n"
		stamp := openStamp
		if strings.Contains(topic, ".database") {
			stamp = closedStamp
		}
		if stamp != "" {
			content += "updated_at: " + stamp + "\n"
		}
		if err := os.WriteFile(filepath.Join(topic, "_meta.yaml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestListShowsBothCopiesWithDividerAndNotice(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("COLUMNS", "80")
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	writeMirroredTopicFixture(t, root, "1793629449341", "Historic WebApp Platform", "2026-11-13T10:45:30Z", "2026-11-13T16:02:11Z")

	output, err := executeCommand(t, "list")
	if err != nil {
		t.Fatalf("list: %v (%s)", err, output)
	}
	// The mirrored topic is listed twice on purpose, newest first.
	if countListRows(output, "1793629449341") != 2 {
		t.Fatalf("list did not show both copies:\n%s", output)
	}
	if !strings.Contains(output, "updated_at=2026-11-13T16:02:11Z") || !strings.Contains(output, "updated_at=2026-11-13T10:45:30Z") {
		t.Fatalf("list did not print both freshness stamps:\n%s", output)
	}
	if !strings.Contains(output, displayDividerToken) {
		t.Fatalf("list did not separate the pair with a divider:\n%s", output)
	}
	if strings.Count(output, "[newer in database]") != 1 {
		t.Fatalf("notice must appear exactly once, on the newer row:\n%s", output)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	newerRow, olderRow := "", ""
	for _, line := range lines {
		if strings.Contains(line, "updated_at=2026-11-13T16:02:11Z") {
			newerRow = line
		}
		if strings.Contains(line, "updated_at=2026-11-13T10:45:30Z") {
			olderRow = line
		}
	}
	if !strings.Contains(newerRow, "[newer in database]") {
		t.Fatalf("notice is not on the newer row: %q", newerRow)
	}
	if strings.Contains(olderRow, "[newer") {
		t.Fatalf("older row carries a notice: %q", olderRow)
	}
	// The newer copy sorts first.
	if strings.Index(output, "16:02:11Z") > strings.Index(output, "10:45:30Z") {
		t.Fatalf("newer copy is not listed first:\n%s", output)
	}
}

func TestListMarksNewerLocallyWhenTheOpenCopyIsNewer(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	writeMirroredTopicFixture(t, root, "1793629449341", "Topic", "2026-11-13T16:02:11Z", "2026-11-13T10:45:30Z")

	output, err := executeCommand(t, "list")
	if err != nil {
		t.Fatalf("list: %v (%s)", err, output)
	}
	if strings.Count(output, "[newer locally]") != 1 {
		t.Fatalf("want exactly one 'newer locally' notice:\n%s", output)
	}
	if strings.Contains(output, "[newer in database]") {
		t.Fatalf("unexpected database notice:\n%s", output)
	}
}

func TestListShowsNoNoticeWhenCopiesShareAStamp(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	stamp := "2026-11-13T10:45:30Z"
	writeMirroredTopicFixture(t, root, "1793629449341", "Topic", stamp, stamp)

	output, err := executeCommand(t, "list")
	if err != nil {
		t.Fatalf("list: %v (%s)", err, output)
	}
	if strings.Contains(output, "[newer") {
		t.Fatalf("identical stamps must not produce a notice:\n%s", output)
	}
	// Two rows still appear: the duplicate itself is the signal.
	if countListRows(output, "1793629449341") != 2 {
		t.Fatalf("want both copies listed:\n%s", output)
	}
}

func TestListJSONExposesNewerCopyAndUpdatedAt(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	writeMirroredTopicFixture(t, root, "1793629449341", "Topic", "2026-11-13T10:45:30Z", "2026-11-13T16:02:11Z")

	output, err := executeCommand(t, "list", "--json")
	if err != nil {
		t.Fatalf("list --json: %v (%s)", err, output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data []struct {
			Storage   string `json:"storage"`
			UpdatedAt string `json:"updated_at"`
			NewerCopy string `json:"newer_copy"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode: %v (%q)", err, output)
	}
	if !response.OK || len(response.Data) != 2 {
		t.Fatalf("list JSON = %#v", response)
	}
	byStorage := map[string]struct {
		UpdatedAt string
		NewerCopy string
	}{}
	for _, item := range response.Data {
		byStorage[item.Storage] = struct{ UpdatedAt, NewerCopy string }{item.UpdatedAt, item.NewerCopy}
	}
	if got := byStorage["closed"]; got.NewerCopy != domain.CopyRegionDatabase || got.UpdatedAt != "2026-11-13T16:02:11Z" {
		t.Fatalf("closed JSON row = %+v", got)
	}
	// JSON carries the plain flag only — no decoration, no caption text.
	if got := byStorage["open"]; got.NewerCopy != "" {
		t.Fatalf("open JSON row = %+v, want empty newer_copy", got)
	}
	if strings.Contains(output, "newer in database") {
		t.Fatalf("JSON leaked presentation text: %q", output)
	}
}

func TestShowPrintsStampAndNoticePerCopy(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	writeMirroredTopicFixture(t, root, "1793629449341", "Topic", "2026-11-13T10:45:30Z", "2026-11-13T16:02:11Z")

	output, err := executeCommand(t, "show", "1793629449341")
	if err != nil {
		t.Fatalf("show: %v (%s)", err, output)
	}
	if !strings.Contains(output, "updated_at=2026-11-13T16:02:11Z") || !strings.Contains(output, "updated_at=2026-11-13T10:45:30Z") {
		t.Fatalf("show did not print both stamps:\n%s", output)
	}
	if strings.Count(output, "[newer in database]") != 1 {
		t.Fatalf("show notice count is wrong:\n%s", output)
	}
}

func TestListNoticeHasNoColorWhenOutputIsPiped(t *testing.T) {
	var buffer bytes.Buffer
	notice := renderNewerCopyNoticeForTest(&buffer, domain.CopyRegionDatabase)
	if strings.Contains(notice, "\x1b[") {
		t.Fatalf("piped output must stay uncoloured: %q", notice)
	}
	if !strings.Contains(notice, "newer in database") {
		t.Fatalf("caption text must survive without a TTY: %q", notice)
	}
}
