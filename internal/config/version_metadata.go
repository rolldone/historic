package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// AppVersion contains application release identity.
type AppVersion struct {
	Name        string
	Code        int
	Workspace   int
	IndexSchema int
}

func CurrentAppVersion() AppVersion {
	return AppVersion{Name: VersionName, Code: VersionCode, Workspace: WorkspaceFormatVersion, IndexSchema: IndexSchemaVersion}
}

func (version AppVersion) Validate() error {
	if !semverPattern.MatchString(version.Name) {
		return fmt.Errorf("invalid version name %q: expected semver", version.Name)
	}
	if version.Code <= 0 {
		return fmt.Errorf("invalid version code %d: must be positive", version.Code)
	}
	if version.Workspace <= 0 {
		return fmt.Errorf("invalid workspace format version %d", version.Workspace)
	}
	if version.IndexSchema <= 0 {
		return fmt.Errorf("invalid index schema version %d", version.IndexSchema)
	}
	return nil
}

// IndexMetadata is the typed historic_meta read model.
type IndexMetadata struct {
	AppVersionName  string
	AppVersionCode  int
	WorkspaceFormat int
	IndexSchema     int
	BuiltAt         string
	BinaryCommit    string
}

func (metadata IndexMetadata) Validate() error {
	if !semverPattern.MatchString(metadata.AppVersionName) {
		return fmt.Errorf("invalid stored app version name %q", metadata.AppVersionName)
	}
	if metadata.AppVersionCode <= 0 || metadata.WorkspaceFormat <= 0 || metadata.IndexSchema <= 0 {
		return fmt.Errorf("stored version metadata contains non-positive values")
	}
	if strings.TrimSpace(metadata.BuiltAt) == "" {
		return fmt.Errorf("stored built_at is empty")
	}
	return nil
}

func parsePositive(value, key string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid %s %q: expected positive integer", key, value)
	}
	return parsed, nil
}
