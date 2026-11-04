package indexer

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"historic/internal/config"
)

var legacyTopicFolderPattern = regexp.MustCompile(`^([0-9]{5})-(.+)$`)

type Diagnosis struct {
	BinaryVersion       string
	ExecutablePath      string
	WorkspaceFormat     int
	CurrentIndexSchema  int
	RequiredIndexSchema int
	Status              string
	MarkdownValid       bool
	IndexExists         bool
	Recommendation      string
	AppVersionName      string
	AppVersionCode      int
	StoredVersionName   string
	StoredVersionCode   int
	CompatibilityAction string
	// DuplicateFileIDs lists FileIDs that appear in more than one topic within
	// the same storage. Since index schema 8 those are legal (FileID is scoped
	// per topic); on an old index they signal a stale/conflicting read model.
	DuplicateFileIDs []DuplicateFileID
}

type DuplicateFileID struct {
	FileID  string                 `json:"file_id"`
	Storage string                 `json:"storage"`
	Topics  []DuplicateFileIDTopic `json:"topics"`
}

type DuplicateFileIDTopic struct {
	TopicID string `json:"topic_id"`
	Path    string `json:"path"`
}

func Diagnose(workspace config.Workspace, binaryVersion string) Diagnosis {
	current := config.CurrentAppVersion()
	diagnosis := Diagnosis{BinaryVersion: binaryVersion, RequiredIndexSchema: current.IndexSchema, MarkdownValid: true, AppVersionName: current.Name, AppVersionCode: current.Code, WorkspaceFormat: current.Workspace}
	diagnosis.ExecutablePath, _ = os.Executable()
	if _, err := os.Stat(workspace.Histories); err != nil {
		diagnosis.Status = "workspace invalid"
		diagnosis.Recommendation = "run historic init in a valid workspace"
		return diagnosis
	}
	diagnosis.DuplicateFileIDs = DetectDuplicateFileIDs(workspace)
	if hasLegacyTopicFolders(workspace) {
		diagnosis.Status = "upgrade required"
		diagnosis.CompatibilityAction = string(config.ActionMigrateWorkspace)
		diagnosis.Recommendation = "run historic upgrade"
		return diagnosis
	}
	if _, err := os.Stat(workspace.Index); errors.Is(err, os.ErrNotExist) {
		diagnosis.Status = "rebuild required"
		diagnosis.Recommendation = "run historic rebuild"
		return diagnosis
	}
	diagnosis.IndexExists = true
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		diagnosis.Status = "rebuild required"
		diagnosis.Recommendation = "run historic rebuild"
		return diagnosis
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		diagnosis.Status = "rebuild required"
		diagnosis.Recommendation = "run historic rebuild"
		return diagnosis
	}
	version, err := schemaVersion(database)
	if err != nil {
		diagnosis.Status = "rebuild required"
		diagnosis.Recommendation = "run historic upgrade to recreate the index schema from Markdown"
		return diagnosis
	}
	diagnosis.CurrentIndexSchema = version
	stored, metaErr := readIndexMetadata(database)
	if metaErr == nil {
		diagnosis.StoredVersionName = stored.AppVersionName
		diagnosis.StoredVersionCode = stored.AppVersionCode
	}
	decision := config.CheckCompatibility(current, stored)
	diagnosis.CompatibilityAction = string(decision.Action)
	switch decision.Action {
	case config.ActionReject:
		diagnosis.Status = "binary too old"
		diagnosis.Recommendation = decision.Reason + "; install a newer Historic binary"
	case config.ActionRebuild:
		diagnosis.Status = "rebuild required"
		diagnosis.Recommendation = "run historic rebuild"
	case config.ActionMigrateWorkspace:
		diagnosis.Status = "upgrade required"
		diagnosis.Recommendation = decision.Reason
	case config.ActionUpdateAppMeta:
		diagnosis.Status = "compatible"
		diagnosis.Recommendation = "metadata will be updated on next rebuild"
	case config.ActionUseExisting:
		diagnosis.Status = "compatible"
		diagnosis.Recommendation = "no action required"
	default:
		switch {
		case version < current.IndexSchema:
			diagnosis.Status = "upgrade required"
			diagnosis.Recommendation = "run historic upgrade"
		case version > current.IndexSchema:
			diagnosis.Status = "binary too old"
			diagnosis.Recommendation = "install a newer Historic binary"
		default:
			diagnosis.Status = "compatible"
			diagnosis.Recommendation = "no action required"
		}
	}
	if len(diagnosis.DuplicateFileIDs) > 0 && diagnosis.Status == "compatible" {
		diagnosis.Recommendation = "no action required; duplicate FileIDs across topics are expected and scoped per topic (run historic rebuild if results look stale)"
	}
	return diagnosis
}

// DetectDuplicateFileIDs reports FileIDs recorded in more than one topic within
// the same storage. It reads Markdown manifests only and never mutates the
// workspace or the index. After index schema 8 this condition is legal: it is
// reported for visibility, not as an error.
func DetectDuplicateFileIDs(workspace config.Workspace) []DuplicateFileID {
	scanned, err := scan(workspace)
	if err != nil {
		return nil
	}
	topicPaths := make(map[string]string, len(scanned.topics))
	for _, topic := range scanned.topics {
		topicPaths[topic.ID.String()+"\x00"+topic.Storage.String()] = topic.Path
	}
	type entry struct {
		topicID string
		path    string
	}
	grouped := make(map[string][]entry)
	order := make([]string, 0)
	seenTopic := make(map[string]map[string]struct{})
	for _, file := range scanned.files {
		if file.FileID.String() == "" || file.Type != "historic_file" {
			continue
		}
		key := file.Storage.String() + "\x00" + file.FileID.String()
		topicID := file.TopicID.String()
		if seenTopic[key] == nil {
			seenTopic[key] = make(map[string]struct{})
		}
		if _, exists := seenTopic[key][topicID]; exists {
			continue
		}
		seenTopic[key][topicID] = struct{}{}
		if len(grouped[key]) == 0 {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], entry{topicID: topicID, path: topicPaths[topicID+"\x00"+file.Storage.String()]})
	}
	duplicates := make([]DuplicateFileID, 0)
	for _, key := range order {
		entries := grouped[key]
		if len(entries) < 2 {
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].topicID < entries[j].topicID })
		storage, fileID, _ := strings.Cut(key, "\x00")
		topics := make([]DuplicateFileIDTopic, 0, len(entries))
		for _, item := range entries {
			topics = append(topics, DuplicateFileIDTopic{TopicID: item.topicID, Path: item.path})
		}
		duplicates = append(duplicates, DuplicateFileID{FileID: fileID, Storage: storage, Topics: topics})
	}
	return duplicates
}

func hasLegacyTopicFolders(workspace config.Workspace) bool {
	for _, root := range []string{workspace.Histories, workspace.Database} {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() && legacyTopicFolderPattern.MatchString(entry.Name()) {
				return true
			}
		}
	}
	return false
}

func Upgrade(workspace config.Workspace) (int, string, error) {
	unlock, err := acquireUpgradeLock(workspace)
	if err != nil {
		return 0, "", err
	}
	defer unlock()
	if _, err := os.Stat(workspace.Histories); err != nil {
		return 0, "", fmt.Errorf("workspace invalid: %w", err)
	}
	count, err := rebuildToTemporaryAndReplace(workspace)
	if err != nil {
		return 0, "", err
	}
	return count, workspace.RelativePath(workspace.Index), nil
}

func rebuildToTemporaryAndReplace(workspace config.Workspace) (int, error) {
	if err := os.MkdirAll(filepath.Dir(workspace.Index), 0o755); err != nil {
		return 0, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(workspace.Index), ".index.sqlite.tmp-")
	if err != nil {
		return 0, err
	}
	temporaryPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return 0, err
	}
	defer os.Remove(temporaryPath)
	count, err := rebuildInto(config.Workspace{Root: workspace.Root, Histories: workspace.Histories, Database: workspace.Database, Index: temporaryPath})
	if err != nil {
		return 0, err
	}
	if err := validateDatabase(temporaryPath); err != nil {
		return 0, err
	}
	backup := workspace.Index + ".backup"
	if _, err := os.Stat(workspace.Index); err == nil {
		if _, err := os.Stat(backup); err == nil {
			_ = os.Remove(backup)
		}
		if err := os.Rename(workspace.Index, backup); err != nil {
			return 0, err
		}
		if err := os.Rename(temporaryPath, workspace.Index); err != nil {
			_ = os.Rename(backup, workspace.Index)
			return 0, err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(temporaryPath, workspace.Index); err != nil {
			return 0, err
		}
	} else {
		return 0, err
	}
	return count, nil
}
