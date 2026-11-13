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

func TestListCollapsesIdenticalCopiesIntoSingleRow(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("COLUMNS", "80")
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	// AC-10: the saved pair carries the same stamp on both sides, which is the
	// real-world case that produced an uninformative duplicate.
	stamp := "2026-11-13T08:53:54Z"
	writeMirroredTopicFixture(t, root, "1794304312863", "Artyplanet Infra", stamp, stamp)

	output, err := executeCommand(t, "list")
	if err != nil {
		t.Fatalf("list: %v (%s)", err, output)
	}
	if strings.Contains(output, "[newer") {
		t.Fatalf("identical stamps must not produce a notice:\n%s", output)
	}
	if strings.Contains(output, displayDividerToken) {
		t.Fatalf("a single row needs no divider:\n%s", output)
	}
	if countListRows(output, "1794304312863") != 1 {
		t.Fatalf("want exactly one row for an identical pair:\n%s", output)
	}
	if !strings.Contains(output, "open") {
		t.Fatalf("the surviving row must be the open copy:\n%s", output)
	}
}

func TestListCollapsesCopiesThatOnlyShareADailyFallbackDate(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("COLUMNS", "80")
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	// Pre-v9 rows carry no stamp: both sides fall back to the same calendar
	// date, which is still "identical" for the purposes of the listing.
	for _, base := range []string{filepath.Join(root, ".historic"), filepath.Join(root, ".historic", ".database")} {
		topic := filepath.Join(base, "1794304312864-legacy")
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		content := "id: \"1794304312864\"\ntitle: Legacy\ncreated: 2026-10-01\nupdated: 2026-11-10\n"
		if err := os.WriteFile(filepath.Join(topic, "_meta.yaml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	output, err := executeCommand(t, "list")
	if err != nil {
		t.Fatalf("list: %v (%s)", err, output)
	}
	if countListRows(output, "1794304312864") != 1 {
		t.Fatalf("identical fallback dates must collapse to one row:\n%s", output)
	}
}

func TestListJSONFlagsTheCollapsedMirroredRow(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	stamp := "2026-11-13T08:53:54Z"
	writeMirroredTopicFixture(t, root, "1794304312863", "Artyplanet Infra", stamp, stamp)

	output, err := executeCommand(t, "list", "--json")
	if err != nil {
		t.Fatalf("list --json: %v (%s)", err, output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data []struct {
			ID        string `json:"id"`
			Storage   string `json:"storage"`
			UpdatedAt string `json:"updated_at"`
			Mirrored  bool   `json:"mirrored"`
			NewerCopy string `json:"newer_copy"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode: %v (%q)", err, output)
	}
	if !response.OK || len(response.Data) != 1 {
		t.Fatalf("list JSON = %#v, want a single collapsed row", response)
	}
	row := response.Data[0]
	// Every pre-existing field keeps its name and value; mirrored is additive.
	if row.ID != "1794304312863" || row.Storage != "open" || row.UpdatedAt != stamp || row.NewerCopy != "" {
		t.Fatalf("list JSON row = %+v, want the open copy", row)
	}
	if !row.Mirrored {
		t.Fatalf("list JSON row = %+v, want mirrored flagged", row)
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

func TestFindRecentCollapsesIdenticalCopiesIntoOneBlock(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("COLUMNS", "120")
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	stamp := "2026-11-13T08:53:54Z"
	writeMirroredTopicFixture(t, root, "1794304312863", "Artyplanet Infra", stamp, stamp)
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "find", "")
	if err != nil {
		t.Fatalf("find: %v (%s)", err, output)
	}
	if strings.Contains(output, displayDividerToken) {
		t.Fatalf("identical copies must not be separated by a divider:\n%s", output)
	}
	if got := strings.Count(output, "1794304312863 ·"); got != 1 {
		t.Fatalf("want exactly one block for an identical pair, got %d:\n%s", got, output)
	}
	if !strings.Contains(output, "[OPEN]") {
		t.Fatalf("the surviving block must be the open copy:\n%s", output)
	}
}

func TestFindRecentSeparatesDivergedCopiesWithDividerAndNotice(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("COLUMNS", "120")
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	// AC-9 regression: different stamps keep both rows, the divider, and the
	// notice on the newer row only.
	writeMirroredTopicFixture(t, root, "1793629449341", "Historic WebApp Platform", "2026-11-13T10:45:30Z", "2026-11-13T16:02:11Z")
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "find", "")
	if err != nil {
		t.Fatalf("find: %v (%s)", err, output)
	}
	if got := strings.Count(output, "1793629449341 ·"); got != 2 {
		t.Fatalf("diverged copies must both be listed, got %d:\n%s", got, output)
	}
	if !strings.Contains(output, displayDividerToken) {
		t.Fatalf("diverged copies must be separated by a divider:\n%s", output)
	}
	if got := strings.Count(output, "newer in database"); got != 1 {
		t.Fatalf("the newer copy must carry the notice exactly once, got %d:\n%s", got, output)
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
