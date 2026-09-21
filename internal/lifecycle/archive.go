package lifecycle

import (
	"fmt"
	"path/filepath"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
)

// Archive is retained as a compatibility wrapper for the explicit close operation.
func (service Service) Archive(id domain.ID) (Change, error) {
	return service.Close(id)
}

func validateTopicPath(workspace config.Workspace, path string) error {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve topic path: %w", err)
	}
	for _, rootPath := range []string{workspace.Histories, workspace.Database} {
		root, rootErr := filepath.EvalSymlinks(rootPath)
		if rootErr != nil {
			continue
		}
		relative, relErr := filepath.Rel(root, realPath)
		if relErr == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil
		}
	}
	return fmt.Errorf("%w: topic path escapes workspace storage roots", domain.ErrConflict)
}
