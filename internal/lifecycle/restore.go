package lifecycle

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/gitproxy"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

// Restore restores one archived topic path from a snapshot into the active
// workspace. It refuses to overwrite an existing active topic unless force is
// explicitly requested.
func Restore(workspace config.Workspace, id domain.ID, snapshot string, force bool) error {
	if !id.Valid() {
		return fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	if strings.TrimSpace(snapshot) == "" {
		return fmt.Errorf("%w: snapshot is empty", domain.ErrConflict)
	}
	repository, err := gitproxy.Open(workspace.Database)
	if err != nil {
		return err
	}
	if _, err := repository.Head(); err != nil {
		return fmt.Errorf("validate snapshot repository: %w", err)
	}
	paths, err := repository.SnapshotPaths(snapshot)
	if err != nil {
		return fmt.Errorf("validate snapshot %s: %w", snapshot, err)
	}
	folder, err := resolveSnapshotFolder(paths, id)
	if err != nil {
		return err
	}
	active, err := activeTopicPath(workspace, id)
	if err == nil && !force {
		return fmt.Errorf("%w: active topic exists at %s; use --force", domain.ErrConflict, workspace.RelativePath(active))
	}
	if err != nil && !os.IsNotExist(err) && !strings.Contains(err.Error(), domain.ErrTopicMissing.Error()) {
		return err
	}
	if err := validateRestorePath(folder); err != nil {
		return err
	}

	staging := filepath.Join(workspace.Histories, ".staging-restore-"+id.String())
	if _, err := os.Lstat(staging); err == nil {
		return fmt.Errorf("%w: restore staging path exists", domain.ErrConflict)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect restore staging: %w", err)
	}
	archive, err := repository.SnapshotArchive(snapshot, folder)
	if err != nil {
		return fmt.Errorf("export snapshot: %w", err)
	}
	if err := extractSnapshotArchive(archive, staging, folder); err != nil {
		_ = os.RemoveAll(staging)
		return fmt.Errorf("stage snapshot: %w", err)
	}
	if err := validateStagedTopic(staging, id); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if active != "" && !force {
		_ = os.RemoveAll(staging)
		return fmt.Errorf("%w: active topic exists at %s; use --force", domain.ErrConflict, workspace.RelativePath(active))
	}
	backup := ""
	if active != "" {
		backup = active + ".restore-backup"
		if err := os.Rename(active, backup); err != nil {
			_ = os.RemoveAll(staging)
			return fmt.Errorf("stage active topic replacement: %w", err)
		}
	}
	if err := os.Rename(filepath.Join(staging, folder), filepath.Join(workspace.Histories, folder)); err != nil {
		if backup != "" {
			_ = os.Rename(backup, active)
		}
		_ = os.RemoveAll(staging)
		return fmt.Errorf("install restored topic: %w", err)
	}
	_ = os.RemoveAll(staging)
	if _, err := indexer.Rebuild(workspace); err != nil {
		installed := filepath.Join(workspace.Histories, folder)
		_ = os.RemoveAll(installed)
		if backup != "" {
			_ = os.Rename(backup, active)
		}
		return fmt.Errorf("rebuild restore index: %w", err)
	}
	if backup != "" {
		_ = os.RemoveAll(backup)
	}
	return nil
}

func resolveSnapshotFolder(paths []string, id domain.ID) (string, error) {
	folders := make(map[string]struct{})
	prefix := id.String() + "-"
	for _, path := range paths {
		parts := strings.Split(filepath.ToSlash(path), "/")
		if len(parts) > 1 && strings.HasPrefix(parts[0], prefix) {
			folders[parts[0]] = struct{}{}
		}
	}
	if len(folders) != 1 {
		return "", fmt.Errorf("%w: snapshot does not contain exactly one topic folder for %s", domain.ErrTopicMissing, id)
	}
	for folder := range folders {
		return folder, nil
	}
	return "", fmt.Errorf("%w: snapshot topic %s", domain.ErrTopicMissing, id)
}

func extractSnapshotArchive(data []byte, destination, folder string) error {
	reader := tar.NewReader(bytes.NewReader(data))
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Clean(header.Name))
		name = strings.TrimPrefix(name, "./")
		if name == "pax_global_header" {
			continue
		}
		if name != folder && !strings.HasPrefix(name, folder+"/") {
			return fmt.Errorf("%w: archive path %q escapes topic folder %q", domain.ErrConflict, name, folder)
		}
		relative := strings.TrimPrefix(name, folder)
		target := filepath.Join(destination, folder, filepath.FromSlash(relative))
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if !header.FileInfo().Mode().IsRegular() {
			return fmt.Errorf("%w: unsupported snapshot entry %s", domain.ErrConflict, name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, header.FileInfo().Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, reader)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func validateStagedTopic(staging string, id domain.ID) error {
	matches, err := filepath.Glob(filepath.Join(staging, "*", markdown.MetaFilename))
	if err != nil || len(matches) != 1 {
		return fmt.Errorf("%w: restored topic metadata is missing", domain.ErrConflict)
	}
	if _, err := markdown.ParseTopicMetadataFile(matches[0]); err != nil {
		return fmt.Errorf("%w: restored topic metadata is invalid: %v", domain.ErrConflict, err)
	}
	return nil
}

func validateRestorePath(path string) error {
	if strings.TrimSpace(path) == "" || filepath.IsAbs(path) || path == "." || strings.Contains(path, "..") || strings.ContainsAny(path, `/\\`) {
		return fmt.Errorf("%w: invalid restore path", domain.ErrConflict)
	}
	return nil
}
