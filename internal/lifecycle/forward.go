package lifecycle

import (
	"fmt"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

// FileTopicID resolves the topic that owns a managed Markdown file.
func FileTopicID(workspace config.Workspace, inputPath string) (domain.ID, error) {
	path, err := resolveMarkdownPath(workspace, inputPath)
	if err != nil {
		return "", err
	}
	id := topicIDFromPath(workspace, path)
	if id == "" {
		return "", fmt.Errorf("%w: cannot resolve topic for %q", domain.ErrTopicMissing, inputPath)
	}
	return id, nil
}

// ForwardFile marks one managed file in an open topic as forwarded to target.
// The file content is never removed or copied; only the frontmatter status and
// the forward destination change. The target topic must extend the source
// topic, otherwise no filesystem change is made.
func ForwardFile(workspace config.Workspace, inputPath string, target domain.ID) (FileChange, error) {
	if !target.Valid() {
		return FileChange{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, target)
	}
	path, err := resolveMarkdownPath(workspace, inputPath)
	if err != nil {
		return FileChange{}, err
	}
	if isInside(workspace.Database, path) {
		return FileChange{}, fmt.Errorf("%w: %q is in closed storage; run \"historic open\" before forwarding", domain.ErrConflict, workspace.RelativePath(path))
	}
	sourceID := topicIDFromPath(workspace, path)
	if sourceID == "" {
		return FileChange{}, fmt.Errorf("%w: cannot resolve topic for %q", domain.ErrTopicMissing, inputPath)
	}
	targetPath, _, err := TopicLocation(workspace, target)
	if err != nil {
		return FileChange{}, err
	}
	meta, err := markdown.ReadTopicMetadata(targetPath)
	if err != nil {
		return FileChange{}, fmt.Errorf("read forward target metadata %s: %w", workspace.RelativePath(targetPath), err)
	}
	if string(meta.Extends) != sourceID.String() {
		return FileChange{}, fmt.Errorf("%w: target topic %s does not extend source topic %s", domain.ErrConflict, target, sourceID)
	}
	return changeFileStatus(workspace, inputPath, domain.StatusForwarded, domain.TopicID(target))
}
