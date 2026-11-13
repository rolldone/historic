package lifecycle

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

// ConflictPathsError reports that a save/close was rejected because the closed
// mirror holds strictly newer content for one or more managed files. It names
// the offending paths so the operator can review the exact files instead of the
// whole topic.
type ConflictPathsError struct {
	ID    domain.ID
	Paths []string
}

func (conflict ConflictPathsError) Error() string {
	return fmt.Sprintf("%v: mirror copy is newer for %s; inspect with \"historic read\" or \"historic git show\" then re-run with --force",
		domain.ErrConflict, strings.Join(conflict.Paths, ", "))
}

// Unwrap lets errors.Is(err, domain.ErrConflict) keep working for callers that
// map conflicts onto exit codes and JSON envelopes.
func (conflict ConflictPathsError) Unwrap() error { return domain.ErrConflict }

// ensureNoNewerMirror rejects the write when a closed mirror holds newer content
// for at least one managed file. The comparison is per file and stays on the
// filesystem: it reads the open and mirror _meta.yaml and hashes each candidate
// file on the fly, so it keeps working after .historic/.index.sqlite is deleted
// (AC-6). It is a no-op when the topic has no mirror yet.
func ensureNoNewerMirror(workspace config.Workspace, id domain.ID) error {
	paths, err := mirroredNewerPaths(workspace, id)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}
	return ConflictPathsError{ID: id, Paths: paths}
}

// mirroredNewerPaths lists the managed paths whose mirror copy is strictly newer
// than the open copy AND whose bytes differ from the open copy. A path is only
// considered when it exists on both sides, and an equal content hash always
// allows the write through (identical content is a no-op, not a conflict).
func mirroredNewerPaths(workspace config.Workspace, id domain.ID) ([]string, error) {
	openPath, storage, err := TopicLocation(workspace, id)
	if err != nil {
		return nil, err
	}
	if storage != domain.StorageOpen {
		return nil, nil
	}
	archive, err := findClosedTopic(workspace, id)
	if err != nil {
		return nil, err
	}
	if archive.path == "" {
		return nil, nil
	}
	openMeta, err := markdown.ReadTopicMetadata(openPath)
	if err != nil {
		return nil, fmt.Errorf("read open topic metadata: %w", err)
	}
	mirrorMeta, err := markdown.ReadTopicMetadata(archive.path)
	if err != nil {
		return nil, fmt.Errorf("read mirror topic metadata: %w", err)
	}

	openStamps := fileStampIndex(openMeta)
	mirrorStamps := fileStampIndex(mirrorMeta)
	openTopicStamp := strings.TrimSpace(openMeta.UpdatedAt)
	mirrorTopicStamp := strings.TrimSpace(mirrorMeta.UpdatedAt)

	paths := make([]string, 0, len(mirrorStamps))
	for path := range mirrorStamps {
		if _, ok := openStamps[path]; !ok {
			// The file is only in the mirror; dropping it is a local deletion,
			// not an edit that the mirror should override.
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)

	conflicts := make([]string, 0)
	for _, path := range paths {
		openStamp := guardStamp(openStamps[path], openTopicStamp, openPath, path)
		mirrorStamp := guardStamp(mirrorStamps[path], mirrorTopicStamp, archive.path, path)
		if openStamp == "" || mirrorStamp == "" || mirrorStamp <= openStamp {
			continue
		}
		openHash, openErr := hashTopicFileIfPresent(filepath.Join(openPath, filepath.FromSlash(path)))
		if openErr != nil {
			return nil, openErr
		}
		mirrorHash, mirrorErr := hashTopicFileIfPresent(filepath.Join(archive.path, filepath.FromSlash(path)))
		if mirrorErr != nil {
			return nil, mirrorErr
		}
		if openHash == mirrorHash {
			continue
		}
		conflicts = append(conflicts, path)
	}
	return conflicts, nil
}

// fileStampIndex maps each managed path to its per-file freshness stamp,
// omitting entries with no stamp so the caller can fall back deterministically.
func fileStampIndex(metadata markdown.TopicMetadata) map[string]string {
	index := make(map[string]string, len(metadata.Files))
	for _, file := range metadata.Files {
		path := strings.TrimSpace(filepath.ToSlash(file.Path))
		stamp := strings.TrimSpace(file.UpdatedAt)
		if path == "" || stamp == "" {
			continue
		}
		index[path] = stamp
	}
	return index
}

// guardStamp resolves the stamp used for one path: the per-file stamp first,
// then the file's own calendar `updated`/`created` date, then the topic stamp.
// An empty result means "no freshness information", which makes the guard skip
// the path rather than risk a false positive.
func guardStamp(stamp, topicStamp, root, path string) string {
	if stamp != "" {
		return stamp
	}
	input, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return topicStamp
	}
	document, err := markdown.Parse(filepath.Join(root, filepath.FromSlash(path)), input)
	if err != nil {
		return topicStamp
	}
	if fallback := documentedStamp(document.Frontmatter); fallback != "" {
		return fallback
	}
	return topicStamp
}

// hashTopicFileIfPresent hashes a managed file, reporting a zero hash when the
// file is absent so a missing copy is always treated as different content.
func hashTopicFileIfPresent(path string) ([sha256.Size]byte, error) {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return [sha256.Size]byte{}, nil
		}
		return [sha256.Size]byte{}, err
	}
	return hashTopicFile(path)
}
