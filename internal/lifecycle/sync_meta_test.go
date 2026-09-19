package lifecycle

import "testing"

func TestSyncMetaSectionPreservesExistingAndAvoidsDuplicates(t *testing.T) {
	body := "# Topic\n\n## Files\n\n- [old.md](./old.md)\n\n## Progress\n\n"
	body = syncMetaSection(body, "## Files", []string{"wos/19-task.md", "old.md"})
	if count := stringsCount(body, "./wos/19-task.md"); count != 1 {
		t.Fatalf("managed link count = %d, body=%q", count, body)
	}
	if count := stringsCount(body, "./old.md"); count != 1 {
		t.Fatalf("existing link count = %d, body=%q", count, body)
	}
	if !containsText(body, "- [old.md](./old.md)") || !containsText(body, "## Progress") {
		t.Fatalf("existing section content lost: %q", body)
	}
	body = syncMetaSection(body, "## Files", []string{"wos/19-task.md"})
	if count := stringsCount(body, "./wos/19-task.md"); count != 1 {
		t.Fatalf("repeated managed link count = %d, body=%q", count, body)
	}
}

func TestSyncMetaSectionAddsAssetsBeforeProgress(t *testing.T) {
	body := "# Topic\n\n## Files\n\n\n## Progress\n\n"
	body = syncMetaSection(body, "## Assets", []string{"image.png", "docs/brief.pdf"})
	if !containsText(body, "## Assets") || !containsText(body, "- [image.png](./image.png)") || !containsText(body, "- [brief.pdf](./docs/brief.pdf)") {
		t.Fatalf("assets section missing: %q", body)
	}
}

func stringsCount(value, needle string) int {
	count := 0
	for len(value) > 0 {
		index := stringsIndex(value, needle)
		if index < 0 {
			return count
		}
		count++
		value = value[index+len(needle):]
	}
	return count
}

func stringsIndex(value, needle string) int {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return index
		}
	}
	return -1
}

func containsText(value, needle string) bool { return stringsIndex(value, needle) >= 0 }
