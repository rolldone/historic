package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/identifier"
	"historic/internal/indexer"
	"historic/internal/markdown"

	"gopkg.in/yaml.v3"
)

var legacyTopicFolderPattern = regexp.MustCompile(`^([0-9]{5})-(.+)$`)

// TopicIDMigrationOptions controls legacy topic identity migration.
type TopicIDMigrationOptions struct {
	DryRun   bool
	LegacyID string
	Clock    identifier.MillisecondClock
}

// TopicIDMigrationMapping describes one legacy folder mapping.
type TopicIDMigrationMapping struct {
	OldID   string `json:"old_id" yaml:"old_id"`
	NewID   string `json:"new_id" yaml:"new_id"`
	OldPath string `json:"old_path" yaml:"old_path"`
	NewPath string `json:"new_path" yaml:"new_path"`
	Action  string `json:"action" yaml:"action"`
}

// TopicIDMigrationReport is the deterministic migration plan/result.
type TopicIDMigrationReport struct {
	DryRun   bool                      `json:"dry_run"`
	Mappings []TopicIDMigrationMapping `json:"mappings"`
}

type legacyTopic struct {
	id       string
	path     string
	root     string
	storage  domain.StorageState
	slug     string
	metadata markdown.TopicMetadata
}

type topicAliasMap struct {
	Aliases map[string]string `yaml:"aliases" json:"aliases"`
}

const topicAliasFilename = ".topic-id-aliases.yaml"

// MigrateTopicIDs scans, plans, and optionally commits legacy topic renames.
func (store TopicStore) MigrateTopicIDs(options TopicIDMigrationOptions) (TopicIDMigrationReport, error) {
	if options.Clock == nil {
		options.Clock = migrationClock{}
	}
	lock := identifier.NewFileLock(filepath.Join(store.Workspace.Histories, ".topic-id.lock"))
	if err := lock.Acquire(context.Background()); err != nil {
		return TopicIDMigrationReport{}, err
	}
	defer lock.Release()

	topics, err := scanLegacyTopics(store.Workspace, options.LegacyID)
	if err != nil {
		return TopicIDMigrationReport{}, err
	}
	plan, err := planTopicIDMigration(store.Workspace, topics, options.Clock)
	if err != nil {
		return TopicIDMigrationReport{}, err
	}
	report := TopicIDMigrationReport{DryRun: options.DryRun, Mappings: plan}
	if options.DryRun {
		return report, nil
	}
	if err := commitTopicIDMigration(store.Workspace, topics, plan); err != nil {
		return TopicIDMigrationReport{}, err
	}
	return report, nil
}

func scanLegacyTopics(workspace config.Workspace, onlyID string) ([]legacyTopic, error) {
	var topics []legacyTopic
	for _, item := range []struct {
		root    string
		storage domain.StorageState
	}{{workspace.Histories, domain.StorageOpen}, {workspace.Database, domain.StorageClosed}} {
		entries, err := os.ReadDir(item.root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("scan migration root: %w", err)
		}
		for _, entry := range entries {
			match := legacyTopicFolderPattern.FindStringSubmatch(entry.Name())
			if len(match) != 3 || !entry.IsDir() || (onlyID != "" && match[1] != onlyID) {
				continue
			}
			path := filepath.Join(item.root, entry.Name())
			metadata, err := markdown.ReadTopicMetadata(path)
			if err != nil {
				return nil, fmt.Errorf("validate legacy topic %s: %w", workspace.RelativePath(path), err)
			}
			if metadata.ID.String() != match[1] {
				return nil, fmt.Errorf("%w: metadata ID %s does not match folder ID %s", domain.ErrConflict, metadata.ID, match[1])
			}
			topics = append(topics, legacyTopic{id: match[1], path: path, root: item.root, storage: item.storage, slug: match[2], metadata: metadata})
		}
	}
	sort.Slice(topics, func(i, j int) bool {
		if topics[i].id != topics[j].id {
			return topics[i].id < topics[j].id
		}
		return filepath.ToSlash(topics[i].path) < filepath.ToSlash(topics[j].path)
	})
	return topics, nil
}

func planTopicIDMigration(workspace config.Workspace, topics []legacyTopic, clock identifier.MillisecondClock) ([]TopicIDMigrationMapping, error) {
	if len(topics) == 0 {
		return []TopicIDMigrationMapping{}, nil
	}
	groups := make([]string, 0)
	seen := make(map[string]struct{})
	for _, topic := range topics {
		if _, ok := seen[topic.id]; !ok {
			seen[topic.id] = struct{}{}
			groups = append(groups, topic.id)
		}
	}
	mappings := make([]TopicIDMigrationMapping, 0, len(topics))
	maxExisting := int64(0)
	for _, root := range []string{workspace.Histories, workspace.Database} {
		entries, _ := os.ReadDir(root)
		for _, entry := range entries {
			if id, ok := domain.TopicIDFromFolder(entry.Name()); ok {
				if value, parseErr := strconv.ParseInt(id.String(), 10, 64); parseErr == nil && value > maxExisting {
					maxExisting = value
				}
			}
		}
	}
	candidate := clock.Now().UTC().UnixMilli()
	if candidate <= maxExisting {
		candidate = maxExisting + 1
	}
	for _, oldID := range groups {
		newID, err := domain.ParseTopicID(strconv.FormatInt(candidate, 10))
		if err != nil {
			return nil, fmt.Errorf("allocate migration ID for %s: %w", oldID, err)
		}
		candidate++
		for _, topic := range topics {
			if topic.id != oldID {
				continue
			}
			newPath := filepath.Join(topic.root, newID.String()+"-"+topic.slug)
			if _, err := os.Lstat(newPath); err == nil {
				return nil, fmt.Errorf("%w: migration target exists %s", domain.ErrConflict, workspace.RelativePath(newPath))
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("inspect migration target: %w", err)
			}
			mappings = append(mappings, TopicIDMigrationMapping{OldID: topic.id, NewID: newID.String(), OldPath: workspace.RelativePath(topic.path), NewPath: workspace.RelativePath(newPath), Action: "rename"})
		}
	}
	return mappings, nil
}

type stagedTopic struct {
	oldPath string
	stage   string
	newPath string
	oldMeta []byte
	newID   domain.TopicID
}

func commitTopicIDMigration(workspace config.Workspace, topics []legacyTopic, mappings []TopicIDMigrationMapping) error {
	if len(mappings) == 0 {
		return nil
	}
	byPath := make(map[string]legacyTopic, len(topics))
	for _, topic := range topics {
		byPath[workspace.RelativePath(topic.path)] = topic
	}
	staged := make([]stagedTopic, 0, len(mappings))
	rollback := func() {
		for index := len(staged) - 1; index >= 0; index-- {
			item := staged[index]
			if _, err := os.Lstat(item.newPath); err == nil {
				_ = os.Rename(item.newPath, item.stage)
			}
			if _, err := os.Lstat(item.stage); err == nil {
				_ = os.Rename(item.stage, item.oldPath)
			}
		}
	}
	for index, mapping := range mappings {
		topic, ok := byPath[mapping.OldPath]
		if !ok {
			rollback()
			return fmt.Errorf("%w: migration source disappeared %s", domain.ErrTopicMissing, mapping.OldPath)
		}
		oldMeta, err := os.ReadFile(filepath.Join(topic.path, markdown.MetaFilename))
		if err != nil {
			rollback()
			return fmt.Errorf("read migration metadata: %w", err)
		}
		stage := filepath.Join(topic.root, fmt.Sprintf(".staging-topic-id-%s-%d", topic.id, index))
		if err := os.Rename(topic.path, stage); err != nil {
			rollback()
			return fmt.Errorf("stage topic %s: %w", mapping.OldPath, err)
		}
		newID, _ := domain.ParseTopicID(mapping.NewID)
		metadata, err := markdown.ReadTopicMetadata(stage)
		if err == nil {
			metadata.ID = domain.ID(newID)
			err = markdown.WriteTopicMetadata(filepath.Join(stage, markdown.MetaFilename), metadata)
		}
		if err != nil {
			_ = os.Rename(stage, topic.path)
			rollback()
			return fmt.Errorf("update migration metadata: %w", err)
		}
		if err := os.Rename(stage, filepath.FromSlash(filepath.Join(workspace.Root, mapping.NewPath))); err != nil {
			_ = os.Rename(stage, topic.path)
			rollback()
			return fmt.Errorf("commit topic rename %s: %w", mapping.OldPath, err)
		}
		staged = append(staged, stagedTopic{oldPath: topic.path, stage: stage, newPath: filepath.FromSlash(filepath.Join(workspace.Root, mapping.NewPath)), oldMeta: oldMeta, newID: newID})
	}
	backupPath := filepath.Join(workspace.Histories, fmt.Sprintf(".topic-id-migration-backup-%d.yaml", time.Now().UTC().UnixNano()))
	if err := writeMigrationBackup(backupPath, mappings); err != nil {
		rollback()
		return err
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		rollback()
		_ = os.Remove(backupPath)
		return fmt.Errorf("rebuild index after topic migration: %w", err)
	}
	if err := writeTopicAliases(workspace, mappings); err != nil {
		rollback()
		_ = os.Remove(backupPath)
		return err
	}
	return nil
}

func writeMigrationBackup(path string, mappings []TopicIDMigrationMapping) error {
	data, err := yaml.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("marshal migration backup: %w", err)
	}
	return atomicMigrationWrite(path, data)
}

func writeTopicAliases(workspace config.Workspace, mappings []TopicIDMigrationMapping) error {
	aliases := topicAliasMap{Aliases: map[string]string{}}
	path := filepath.Join(workspace.Histories, topicAliasFilename)
	if input, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(input, &aliases)
		if aliases.Aliases == nil {
			aliases.Aliases = map[string]string{}
		}
	}
	for _, mapping := range mappings {
		aliases.Aliases[mapping.OldID] = mapping.NewID
	}
	data, err := yaml.Marshal(aliases)
	if err != nil {
		return fmt.Errorf("marshal topic aliases: %w", err)
	}
	return atomicMigrationWrite(path, data)
}

func atomicMigrationWrite(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".topic-id-migration-*.tmp")
	if err != nil {
		return fmt.Errorf("create migration temporary file: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	return nil
}

type migrationClock struct{}

func (migrationClock) Now() time.Time { return time.Now() }

// MarshalJSON keeps migration reports stable for CLI consumers.
func (report TopicIDMigrationReport) MarshalJSON() ([]byte, error) {
	type alias TopicIDMigrationReport
	return json.Marshal(alias(report))
}
