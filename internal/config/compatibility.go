package config

import "fmt"

type CompatibilityAction string

const (
	ActionUseExisting      CompatibilityAction = "use_existing"
	ActionUpdateAppMeta    CompatibilityAction = "update_app_meta"
	ActionRebuild          CompatibilityAction = "rebuild"
	ActionMigrateWorkspace CompatibilityAction = "migrate_workspace"
	ActionReject           CompatibilityAction = "reject"
)

type CompatibilityDecision struct {
	Action         CompatibilityAction
	Reason         string
	BackupRequired bool
}

func CheckCompatibility(current AppVersion, stored IndexMetadata) CompatibilityDecision {
	if stored.IndexSchema < current.IndexSchema {
		return CompatibilityDecision{Action: ActionRebuild, Reason: fmt.Sprintf("index schema %d is older than required %d", stored.IndexSchema, current.IndexSchema), BackupRequired: true}
	}
	if stored.IndexSchema > current.IndexSchema {
		return CompatibilityDecision{Action: ActionReject, Reason: fmt.Sprintf("index schema %d is newer than supported %d; upgrade binary", stored.IndexSchema, current.IndexSchema)}
	}
	if stored.WorkspaceFormat > current.Workspace {
		return CompatibilityDecision{Action: ActionReject, Reason: fmt.Sprintf("workspace format %d is newer than supported %d; upgrade binary", stored.WorkspaceFormat, current.Workspace)}
	}
	if stored.WorkspaceFormat < current.Workspace {
		return CompatibilityDecision{Action: ActionMigrateWorkspace, Reason: fmt.Sprintf("workspace format %d is older than current %d", stored.WorkspaceFormat, current.Workspace)}
	}
	if stored.AppVersionCode != current.Code || stored.AppVersionName != current.Name {
		return CompatibilityDecision{Action: ActionUpdateAppMeta, Reason: "app version changed but schema is compatible"}
	}
	return CompatibilityDecision{Action: ActionUseExisting, Reason: "all versions are compatible"}
}
