package lifecycle

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"

	_ "modernc.org/sqlite"
)

// BumpResult reports the outcome of the content-change bump pass. BumpedFiles
// lists the relative paths whose frontmatter `updated` date was advanced;
// Warnings lists per-file (or per-topic) problems that did not abort the pass,
// so callers can surface them without failing the surrounding operation.
type BumpResult struct {
	BumpedFiles []string
	Warnings    []string
}

// Count returns the number of files written by the pass.
func (result BumpResult) Count() int { return len(result.BumpedFiles) }

// BumpChangedFiles advances the frontmatter `updated` date of every managed
// Markdown file in the given OPEN topics whose on-disk content hash differs
// from the digest stored in the index read model (files.hash). It reuses
// markdown.Parse/Write so the write shape matches `historic status` and
// laterDate so `updated` never moves behind `created`.
//
// The pass is deliberately conservative: it only writes when it can prove a
// change against an indexed baseline, so a missing or stale index never causes
// a workspace-wide rewrite of `updated`. Assets, non-Historic Markdown, and
// _meta.yaml are never touched, and the closed mirror (.database) is out of
// scope because it is produced by save/mirror, never edited in place.
//
// A single unreadable or invalid file becomes a warning while the pass keeps
// going; the index is rebuilt once at the end so the new hashes are stored.
func BumpChangedFiles(workspace config.Workspace, ids []domain.ID) BumpResult {
	result := BumpResult{BumpedFiles: make([]string, 0), Warnings: make([]string, 0)}
	if len(ids) == 0 {
		return result
	}
	baseline, err := indexedOpenHashes(workspace)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("bump updated: skipped (index baseline unavailable): %v", err))
		return result
	}
	today := fileStatusNow().UTC().Format("2006-01-02")
	unique := make([]domain.ID, 0, len(ids))
	seen := make(map[domain.ID]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i].String() < unique[j].String() })
	for _, id := range unique {
		path, storage, locErr := TopicLocation(workspace, id)
		if locErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("bump updated: %s: %v", id, locErr))
			continue
		}
		if storage != domain.StorageOpen {
			continue
		}
		bumpTopicFiles(workspace, id, path, today, baseline, &result)
	}
	if result.Count() == 0 {
		return result
	}
	if _, rebuildErr := indexer.Rebuild(workspace); rebuildErr != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("bump updated: index rebuild failed: %v", rebuildErr))
	}
	return result
}

// BumpAllChangedFiles runs the bump pass across every open topic.
func BumpAllChangedFiles(workspace config.Workspace) BumpResult {
	return BumpChangedFiles(workspace, openTopicIDs(workspace))
}

// openTopicIDs lists the topic IDs currently in open storage. Discovery never
// fails: an unreadable root yields no IDs so the pass simply does nothing.
func openTopicIDs(workspace config.Workspace) []domain.ID {
	entries, err := os.ReadDir(workspace.Histories)
	if err != nil {
		return nil
	}
	ids := make([]domain.ID, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !syncTopicFolderPattern.MatchString(entry.Name()) {
			continue
		}
		separator := strings.Index(entry.Name(), "-")
		if separator <= 0 {
			continue
		}
		id, idErr := domain.ParseTopicIdentity(entry.Name()[:separator])
		if idErr != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

// indexedOpenHashes loads the stored content digest of every open-storage file,
// keyed by topic ID and POSIX topic-relative path.
func indexedOpenHashes(workspace config.Workspace) (map[string]string, error) {
	if _, err := os.Stat(workspace.Index); err != nil {
		return nil, fmt.Errorf("index %s: %w", workspace.RelativePath(workspace.Index), err)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}
	defer database.Close()
	rows, err := database.Query(`SELECT topic_id, path, hash FROM files WHERE storage = ?`, domain.StorageOpen.String())
	if err != nil {
		return nil, fmt.Errorf("read files.hash: %w", err)
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var topicID, path, hash string
		if err := rows.Scan(&topicID, &path, &hash); err != nil {
			return nil, fmt.Errorf("scan files.hash: %w", err)
		}
		result[bumpBaselineKey(topicID, path)] = hash
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate files.hash: %w", err)
	}
	return result, nil
}

func bumpBaselineKey(topicID, path string) string {
	return topicID + "\x00" + path
}

// bumpTopicFiles walks one open topic and advances `updated` on managed files
// whose content changed. Walk errors and unreadable entries degrade to
// warnings so one bad file cannot fail the whole pass.
func bumpTopicFiles(workspace config.Workspace, id domain.ID, topicPath, today string, baseline map[string]string, result *BumpResult) {
	walkErr := filepath.WalkDir(topicPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("bump updated: %s: %v", workspace.RelativePath(path), err))
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("bump updated: skipped symlink %s", workspace.RelativePath(path)))
			return nil
		}
		base := filepath.Base(path)
		if !strings.EqualFold(filepath.Ext(path), ".md") || base == markdown.MetaFilename || base == "_meta.md" {
			return nil
		}
		relative, relErr := filepath.Rel(topicPath, path)
		if relErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("bump updated: %s: %v", workspace.RelativePath(path), relErr))
			return nil
		}
		stored, indexed := baseline[bumpBaselineKey(id.String(), filepath.ToSlash(relative))]
		if !indexed {
			// No baseline: a brand-new file, or the index predates it. Never
			// guess — skipping keeps the pass from rewriting unindexed files.
			return nil
		}
		current := indexer.HashFile(path)
		if current == "" || current == stored {
			return nil
		}
		if bumpErr := bumpFileDate(workspace, path, today, result); bumpErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("bump updated: %s: %v", workspace.RelativePath(path), bumpErr))
		}
		return nil
	})
	if walkErr != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("bump updated: %s: %v", workspace.RelativePath(topicPath), walkErr))
	}
}

// bumpFileDate rewrites one file's `updated` frontmatter field. It writes only
// when the value actually advances, keeping the pass idempotent (a repeated run
// with unchanged content performs no write) and never moving `updated`
// backwards. Non-Historic Markdown is left untouched.
func bumpFileDate(workspace config.Workspace, path, today string, result *BumpResult) error {
	input, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	document, err := markdown.Parse(path, input)
	if err != nil {
		if markdown.LooksLikeHistoricFile(input) {
			return fmt.Errorf("invalid Historic Markdown: %w", err)
		}
		// A plain asset or non-Historic Markdown file: nothing to bump.
		return nil
	}
	next := laterDate(document.Frontmatter.Created, today)
	if next == "" || next <= document.Frontmatter.Updated {
		return nil
	}
	document.Frontmatter.Updated = next
	if err := markdown.WriteFile(path, document); err != nil {
		return err
	}
	result.BumpedFiles = append(result.BumpedFiles, workspace.RelativePath(path))
	return nil
}
