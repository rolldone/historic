package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateWorkspaceReadinessMatrix(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string, workspace Workspace)
		want  ReadinessCode
	}{
		{
			name:  "missing historic",
			setup: func(t *testing.T, root string, workspace Workspace) {},
			want:  ReadinessNotHistoricWorkspace,
		},
		{
			name: "legacy conflict",
			setup: func(t *testing.T, root string, workspace Workspace) {
				t.Helper()
				if err := os.Mkdir(workspace.Histories, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(root, LegacyHistoriesDirName), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			want: ReadinessLegacyConflict,
		},
		{
			name: "missing index",
			setup: func(t *testing.T, root string, workspace Workspace) {
				t.Helper()
				if err := os.MkdirAll(workspace.Histories, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(workspace.Database, ".git"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			want: ReadinessMissingIndex,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			workspace := NewWorkspace(root)
			test.setup(t, root, workspace)
			before := snapshot(t, root)
			result, err := ValidateWorkspace(workspace)
			if result.State != test.want {
				t.Fatalf("state = %q, want %q (err=%v)", result.State, test.want, err)
			}
			var readiness *ReadinessError
			if !errors.As(err, &readiness) || readiness.Code != test.want {
				t.Fatalf("error = %#v, want typed code %q", err, test.want)
			}
			if after := snapshot(t, root); before != after {
				t.Fatalf("validation changed filesystem: before=%q after=%q", before, after)
			}
		})
	}
}

func snapshot(t *testing.T, root string) string {
	t.Helper()
	entries := make([]string, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		entries = append(entries, path+":"+info.Mode().String())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return join(entries)
}

func join(values []string) string {
	result := ""
	for _, value := range values {
		result += value + "\n"
	}
	return result
}
