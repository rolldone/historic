package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/identifier"
	"historic/internal/markdown"

	"gopkg.in/yaml.v3"
)

// FileIDMigrationOptions controls legacy managed-file ID migration.
type FileIDMigrationOptions struct {
	DryRun  bool
	TopicID *domain.ID
	Storage *domain.StorageState
}

// FileIDMigrationReport summarizes one migration plan or commit.
type FileIDMigrationReport struct {
	DryRun           bool            `json:"dry_run"`
	TopicsScanned    int             `json:"topics_scanned"`
	ManifestsUpdated int             `json:"manifests_updated"`
	FilesMigrated    int             `json:"files_migrated"`
	FilesPreserved   int             `json:"files_preserved"`
	FilesSkipped     int             `json:"files_skipped"`
	Mappings         []FileIDMapping `json:"mappings"`
}

// FileIDMapping records one preserved or assigned managed-file ID.
type FileIDMapping struct {
	TopicID  domain.ID     `json:"topic_id"`
	Path     string        `json:"path"`
	LegacyID string        `json:"legacy_id"`
	FileID   domain.FileID `json:"file_id"`
	Action   string        `json:"action"`
}

type migrationPlan struct {
	path     string
	metadata markdown.TopicMetadata
	before   []byte
}

// NewServiceWithFileIDGenerator creates a lifecycle service with an injectable
// generator for migration tests and deterministic callers.
func NewServiceWithFileIDGenerator(workspace config.Workspace, generator identifier.Generator) Service {
	service := NewService(workspace)
	service.FileIDGenerator = generator
	return service
}

// MigrateFileIDs assigns UUIDv7 IDs to legacy manifest entries atomically.
func (service Service) MigrateFileIDs(options FileIDMigrationOptions) (FileIDMigrationReport, error) {
	if options.Storage != nil && !options.Storage.IsValid() {
		return FileIDMigrationReport{}, fmt.Errorf("%w: invalid storage scope %q", domain.ErrConflict, *options.Storage)
	}
	if options.TopicID != nil && !options.TopicID.Valid() {
		return FileIDMigrationReport{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, *options.TopicID)
	}
	paths, err := migrationTopicPaths(service.Workspace, options)
	if err != nil {
		return FileIDMigrationReport{}, err
	}
	plans, report, err := service.planFileIDMigration(paths, options.DryRun)
	if err != nil {
		return FileIDMigrationReport{}, err
	}
	if options.DryRun || len(plans) == 0 {
		return report, nil
	}
	if err := commitFileIDMigration(service.Workspace, plans); err != nil {
		return FileIDMigrationReport{}, err
	}
	report.ManifestsUpdated = len(plans)
	return report, nil
}

func migrationTopicPaths(workspace config.Workspace, options FileIDMigrationOptions) ([]string, error) {
	paths := make([]string, 0)
	roots := []struct {
		path    string
		storage domain.StorageState
	}{{workspace.Histories, domain.StorageOpen}, {workspace.Database, domain.StorageClosed}}
	for _, root := range roots {
		if options.Storage != nil && *options.Storage != root.storage {
			continue
		}
		entries, err := os.ReadDir(root.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", root.path, err)
		}
		for _, entry := range entries {
			match := syncTopicFolderPattern.FindStringSubmatch(entry.Name())
			if len(match) != 3 || !entry.IsDir() {
				continue
			}
			id, parseErr := domain.ParseID(match[1])
			if parseErr != nil || options.TopicID != nil && id != *options.TopicID {
				continue
			}
			paths = append(paths, filepath.Join(root.path, entry.Name()))
		}
	}
	sort.Slice(paths, func(i, j int) bool { return filepath.ToSlash(paths[i]) < filepath.ToSlash(paths[j]) })
	return paths, nil
}

func (service Service) planFileIDMigration(paths []string, dryRun bool) ([]migrationPlan, FileIDMigrationReport, error) {
	report := FileIDMigrationReport{DryRun: dryRun, Mappings: make([]FileIDMapping, 0)}
	used := make(map[domain.FileID]string)
	plans := make([]migrationPlan, 0)
	for _, path := range paths {
		metaPath := filepath.Join(path, markdown.MetaFilename)
		input, err := os.ReadFile(metaPath)
		if err != nil {
			return nil, report, fmt.Errorf("read metadata %s: %w", metaPath, err)
		}
		metadata, err := markdown.ParseTopicMetadataLenient(metaPath, input)
		if err != nil {
			return nil, report, err
		}
		report.TopicsScanned++
		changed := false
		for index := range metadata.Files {
			file := &metadata.Files[index]
			if file.ID.Valid() {
				identity := metadata.ID.String() + "\x00" + file.Path
				if previous, exists := used[file.ID]; exists && previous != identity {
					return nil, report, fmt.Errorf("%w: duplicate file ID %s at %s and %s", domain.ErrConflict, file.ID, previous, identity)
				}
				used[file.ID] = identity
				report.FilesPreserved++
				report.Mappings = append(report.Mappings, FileIDMapping{TopicID: metadata.ID, Path: file.Path, FileID: file.ID, Action: "preserve"})
				continue
			}
			legacyID := file.ID.String()
			fileID, generateErr := service.FileIDGenerator.New()
			if generateErr != nil {
				return nil, report, fmt.Errorf("generate FileID for %s: %w", file.Path, generateErr)
			}
			if _, exists := used[fileID]; exists {
				return nil, report, fmt.Errorf("%w: generated FileID %s", domain.ErrConflict, fileID)
			}
			used[fileID] = filepath.ToSlash(filepath.Join(filepath.Base(path), file.Path))
			file.ID = fileID
			changed = true
			report.FilesMigrated++
			report.Mappings = append(report.Mappings, FileIDMapping{TopicID: metadata.ID, Path: file.Path, LegacyID: legacyID, FileID: fileID, Action: "assign"})
		}
		if changed {
			plans = append(plans, migrationPlan{path: metaPath, metadata: metadata, before: input})
		}
	}
	sort.Slice(report.Mappings, func(i, j int) bool {
		if report.Mappings[i].TopicID != report.Mappings[j].TopicID {
			return report.Mappings[i].TopicID < report.Mappings[j].TopicID
		}
		return report.Mappings[i].Path < report.Mappings[j].Path
	})
	return plans, report, nil
}

func commitFileIDMigration(workspace config.Workspace, plans []migrationPlan) error {
	lock := filepath.Join(workspace.Histories, ".migrate-file-ids.lock")
	file, err := os.OpenFile(lock, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: migration already in progress", domain.ErrConflict)
		}
		return fmt.Errorf("create migration lock: %w", err)
	}
	_ = file.Close()
	defer os.Remove(lock)
	for _, plan := range plans {
		current, readErr := os.ReadFile(plan.path)
		if readErr != nil {
			return readErr
		}
		if string(current) != string(plan.before) {
			return fmt.Errorf("%w: metadata changed during migration: %s", domain.ErrConflict, plan.path)
		}
	}
	backups := make(map[string][]byte, len(plans))
	committed := make([]string, 0, len(plans))
	defer func() {
		for _, path := range committed {
			_ = os.WriteFile(path, backups[path], 0o644)
		}
	}()
	for _, plan := range plans {
		backups[plan.path] = plan.before
		if err := writeMigrationMetadata(plan.path, plan.metadata); err != nil {
			return err
		}
		committed = append(committed, plan.path)
	}
	committed = nil
	return nil
}

func writeMigrationMetadata(path string, metadata markdown.TopicMetadata) error {
	data, err := yamlMarshalMetadata(metadata)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".historic-migrate-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary metadata: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("install migrated metadata: %w", err)
	}
	return nil
}

func yamlMarshalMetadata(metadata markdown.TopicMetadata) ([]byte, error) {
	data, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal migrated metadata: %w", err)
	}
	if _, err := markdown.ParseTopicMetadata("migrated metadata", data); err != nil {
		return nil, fmt.Errorf("validate migrated metadata: %w", err)
	}
	return data, nil
}
