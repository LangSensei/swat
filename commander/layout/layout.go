package layout

import (
	"os"
	"path/filepath"
)

var root = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".swat")
}()

// --- Root ---

// Root returns the swat root directory (~/.swat).
func Root() string { return root }

// --- Blueprints (templates) ---

// BlueprintDir returns the blueprints base directory.
func BlueprintDir() string { return filepath.Join(root, "blueprints") }

// BlueprintSquadsDir returns the blueprints/squads directory.
func BlueprintSquadsDir() string { return filepath.Join(BlueprintDir(), "squads") }

// BlueprintSquadDir returns the blueprint directory for a specific squad.
func BlueprintSquadDir(squad string) string { return filepath.Join(BlueprintSquadsDir(), squad) }

// BlueprintFrameworkDir returns the _framework blueprint directory.
func BlueprintFrameworkDir() string { return BlueprintSquadDir("_framework") }

// BlueprintSkillsDir returns the blueprints/skills directory.
func BlueprintSkillsDir() string { return filepath.Join(BlueprintDir(), "skills") }

// BlueprintMCPsDir returns the blueprints/mcps directory.
func BlueprintMCPsDir() string { return filepath.Join(BlueprintDir(), "mcps") }

// OperationTemplatePath returns the path to the OPERATION.md template.
func OperationTemplatePath() string { return filepath.Join(BlueprintDir(), "OPERATION.md") }

// --- Runtime (squads + operations) ---

// SquadsDir returns the runtime squads base directory.
func SquadsDir() string { return filepath.Join(root, "squads") }

// SquadDir returns the runtime directory for a squad.
func SquadDir(squad string) string { return filepath.Join(SquadsDir(), squad) }

// OperationsDir returns the operations directory for a squad.
func OperationsDir(squad string) string { return filepath.Join(SquadDir(squad), "operations") }

// OperationDir returns the directory for a specific operation.
func OperationDir(squad, opID string) string { return filepath.Join(OperationsDir(squad), opID) }

// OperationMDPath returns the OPERATION.md path for an operation.
func OperationMDPath(squad, opID string) string {
	return filepath.Join(OperationDir(squad, opID), "OPERATION.md")
}

// --- Unclassified (special squad) ---

// UnclassifiedOperationsDir returns the base directory for unclassified operations.
func UnclassifiedOperationsDir() string { return OperationsDir("_unclassified") }

// UnclassifiedOperationDir returns the directory for a specific unclassified operation.
func UnclassifiedOperationDir(opID string) string { return OperationDir("_unclassified", opID) }

// UnclassifiedOperationMDPath returns the OPERATION.md path for an unclassified operation.
func UnclassifiedOperationMDPath(opID string) string { return OperationMDPath("_unclassified", opID) }

// --- Schedules ---

// SchedulesDir returns the schedules directory.
func SchedulesDir() string { return filepath.Join(root, "schedules") }

// --- Init ---

// EnsureDirs creates the standard directory structure.
func EnsureDirs() {
	for _, dir := range []string{
		root,
		BlueprintDir(),
		BlueprintSquadsDir(),
		BlueprintSkillsDir(),
		BlueprintMCPsDir(),
		SquadsDir(),
		UnclassifiedOperationsDir(),
	} {
		os.MkdirAll(dir, 0755)
	}
}
