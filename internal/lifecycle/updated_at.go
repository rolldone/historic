package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/identifier"
	"historic/internal/markdown"
)

// This file owns the high-resolution freshness stamps persisted in _meta.yaml.
// Two stamps exist per topic: a topic-level stamp that advances whenever the
// topic is touched, and a per-file stamp that advances only when that specific
// file changes. The per-file stamp is what makes ordering by "when was this file
// last edited" work (issue #2); a topic-only stamp cannot detect a single edited
// file. Both live in the manifest, so they travel with the mirror and a pull.

// stampClock is injectable so tests can pin the recorded instant.
var stampClock = func() time.Time { return time.Now() }

// stampAt returns the canonical RFC3339 UTC second-precision value.
func stampAt(at time.Time) string { return markdown.FormatTimestamp(at) }

// advanceStamp keeps stamps monotonic: a candidate is only accepted when it is
// strictly newer than the stored value.
func advanceStamp(current string, at time.Time) string {
	next := stampAt(at)
	if next <= current {
		return current
	}
	return next
}

// AdvanceStamp is the exported form of advanceStamp. It returns the canonical
// stamp for at, but never moves a value backwards: when the stored value is
// already at or ahead of at, the stored value is returned unchanged. Callers in
// other packages (for example the repository add path) use it to keep the
// per-file and topic freshness stamps monotonic.
func AdvanceStamp(current string, at time.Time) string {
	return advanceStamp(current, at)
}

// StampTopicFiles advances the topic stamp and the stamp of every managed file
// listed in relativePaths, writing _meta.yaml once. Files not listed keep their
// existing stamp, which is what isolates a one-file edit from the rest of the
// topic. It reports whether the manifest bytes changed.
func StampTopicFiles(workspace config.Workspace, topicPath string, relativePaths []string, at time.Time) (bool, error) {
	metaPath := filepath.Join(topicPath, markdown.MetaFilename)
	metadata, err := markdown.ReadTopicMetadata(topicPath)
	if err != nil {
		return false, fmt.Errorf("read topic metadata: %w", err)
	}
	changed := false
	if next := advanceStamp(metadata.UpdatedAt, at); next != metadata.UpdatedAt {
		metadata.UpdatedAt, changed = next, true
	}
	wanted := make(map[string]struct{}, len(relativePaths))
	for _, relative := range relativePaths {
		wanted[filepath.ToSlash(relative)] = struct{}{}
	}
	for index := range metadata.Files {
		if _, ok := wanted[metadata.Files[index].Path]; !ok {
			continue
		}
		next := advanceStamp(metadata.Files[index].UpdatedAt, at)
		if next == metadata.Files[index].UpdatedAt {
			continue
		}
		metadata.Files[index].UpdatedAt = next
		changed = true
	}
	if !changed {
		return false, nil
	}
	if err := markdown.WriteTopicMetadata(metaPath, metadata); err != nil {
		return false, fmt.Errorf("write topic metadata: %w", err)
	}
	return true, nil
}

// documentedStamp renders the calendar dates a document declares as a
// second-precision stamp, preferring `updated` and falling back to `created`. It
// is used when a file has no stamp yet and its identity cannot supply one
// without inventing freshness (a UUIDv7 minted during the same rebuild). An
// empty result means "no stamp", so ordering falls back to the calendar date.
func documentedStamp(frontmatter domain.Frontmatter) string {
	for _, candidate := range []string{frontmatter.Updated, frontmatter.Created} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if instant, err := time.Parse("2006-01-02", candidate); err == nil {
			return stampAt(instant)
		}
	}
	return ""
}

// BackfillTopicStamps fills empty stamps on every open and closed topic without
// overwriting existing ones. Topic stamps come from the epoch-millisecond topic
// identity, file stamps from the UUIDv7 file identity. Legacy five-digit topics
// carry no instant, so their stamp stays empty and ordering falls back to the
// calendar `updated` date. It reports how many manifests were written.
func BackfillTopicStamps(workspace config.Workspace) (int, error) {
	paths, err := allTopicPaths(workspace)
	if err != nil {
		return 0, err
	}
	written := 0
	for _, topicPath := range paths {
		changed, backfillErr := backfillTopicStampsAt(topicPath)
		if backfillErr != nil {
			if errors.Is(backfillErr, os.ErrNotExist) {
				continue
			}
			return written, backfillErr
		}
		if changed {
			written++
		}
	}
	return written, nil
}

func backfillTopicStampsAt(topicPath string) (bool, error) {
	metadata, err := markdown.ReadTopicMetadata(topicPath)
	if err != nil {
		return false, fmt.Errorf("%s: %w", filepath.Join(topicPath, markdown.MetaFilename), err)
	}
	changed := false
	if metadata.UpdatedAt == "" {
		if instant, ok := identifier.TopicIDTime(metadata.ID); ok {
			metadata.UpdatedAt = stampAt(instant)
			changed = true
		}
	}
	for index := range metadata.Files {
		if metadata.Files[index].UpdatedAt != "" {
			continue
		}
		if instant, ok := identifier.FileIDTime(metadata.Files[index].ID); ok {
			metadata.Files[index].UpdatedAt = stampAt(instant)
			changed = true
		}
	}
	if !changed {
		return false, nil
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topicPath, markdown.MetaFilename), metadata); err != nil {
		return false, err
	}
	return true, nil
}
