package config

import "testing"

func TestCheckCompatibilityRebuildOnOldSchema(t *testing.T) {
	current := AppVersion{Name: "0.3.0", Code: 3, Workspace: 1, IndexSchema: 3}
	stored := IndexMetadata{AppVersionName: "0.2.0", AppVersionCode: 2, WorkspaceFormat: 1, IndexSchema: 2}
	decision := CheckCompatibility(current, stored)
	if decision.Action != ActionRebuild {
		t.Fatalf("action = %q, want rebuild", decision.Action)
	}
	if !decision.BackupRequired {
		t.Fatal("backup should be required")
	}
}

func TestCheckCompatibilityRejectOnNewerSchema(t *testing.T) {
	current := AppVersion{Name: "0.3.0", Code: 3, Workspace: 1, IndexSchema: 3}
	stored := IndexMetadata{AppVersionName: "0.2.0", AppVersionCode: 2, WorkspaceFormat: 1, IndexSchema: 4}
	decision := CheckCompatibility(current, stored)
	if decision.Action != ActionReject {
		t.Fatalf("action = %q, want reject", decision.Action)
	}
}

func TestCheckCompatibilityUpdateAppMeta(t *testing.T) {
	current := AppVersion{Name: "0.3.0", Code: 3, Workspace: 1, IndexSchema: 3}
	stored := IndexMetadata{AppVersionName: "0.2.0", AppVersionCode: 2, WorkspaceFormat: 1, IndexSchema: 3}
	decision := CheckCompatibility(current, stored)
	if decision.Action != ActionUpdateAppMeta {
		t.Fatalf("action = %q, want update_app_meta", decision.Action)
	}
}

func TestCheckCompatibilityUseExisting(t *testing.T) {
	current := AppVersion{Name: "0.3.0", Code: 3, Workspace: 1, IndexSchema: 3}
	stored := IndexMetadata{AppVersionName: "0.3.0", AppVersionCode: 3, WorkspaceFormat: 1, IndexSchema: 3}
	decision := CheckCompatibility(current, stored)
	if decision.Action != ActionUseExisting {
		t.Fatalf("action = %q, want use_existing", decision.Action)
	}
}

func TestAppVersionValidateRejectsEmptyName(t *testing.T) {
	if err := (AppVersion{Name: "", Code: 3, Workspace: 1, IndexSchema: 3}).Validate(); err == nil {
		t.Fatal("empty name should be rejected")
	}
}

func TestAppVersionValidateRejectsZeroCode(t *testing.T) {
	if err := (AppVersion{Name: "0.3.0", Code: 0, Workspace: 1, IndexSchema: 3}).Validate(); err == nil {
		t.Fatal("zero code should be rejected")
	}
}

func TestIndexMetadataValidateRejectsMissingBuiltAt(t *testing.T) {
	if err := (IndexMetadata{AppVersionName: "0.3.0", AppVersionCode: 3, WorkspaceFormat: 1, IndexSchema: 3}).Validate(); err == nil {
		t.Fatal("empty built_at should be rejected")
	}
}
