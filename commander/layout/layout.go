package layout

import (
	"os"
	"path/filepath"
)

var root = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".swat")
}()

// Root returns the swat root directory.
func Root() string { return root }

// --- Blueprint (templates) ---

// BlueprintDir returns the blueprints base directory.
func BlueprintDir() string { return filepath.Join(root, "blueprints") }

// BlueprintSquadsDir returns the blueprints/squads directory.
func BlueprintSquadsDir() string { return filepath.Join(root, "blueprints", "squads") }

// BlueprintSquadDir returns the blueprint directory for a specific squad.
func BlueprintSquadDir(squad string) string {
	return filepath.Join(root, "blueprints", "squads", squad)
}

// BlueprintFrameworkDir returns the _framework blueprint directory.
func BlueprintFrameworkDir() string {
	return filepath.Join(root, "blueprints", "squads", "_framework")
}

// BlueprintSkillsDir returns the blueprints/skills directory.
func BlueprintSkillsDir() string { return filepath.Join(root, "blueprints", "skills") }

// BlueprintMCPsDir returns the blueprints/mcps directory.
func BlueprintMCPsDir() string { return filepath.Join(root, "blueprints", "mcps") }

// --- Runtime ---

// SquadDir returns the runtime directory for a squad.
func SquadDir(squad string) string { return filepath.Join(root, "squads", squad) }

// UnclassifiedOperationsDir returns the base directory for unclassified operations.
func UnclassifiedOperationsDir() string {
	return filepath.Join(root, "squads", "_unclassified", "operations")
}

// UnclassifiedOperationDir returns the directory for a specific unclassified operation.
func UnclassifiedOperationDir(opID string) string {
	return filepath.Join(root, "squads", "_unclassified", "operations", opID)
}

// UnclassifiedOperationMDPath returns the OPERATION.md path for an unclassified operation.
func UnclassifiedOperationMDPath(opID string) string {
	return filepath.Join(UnclassifiedOperationDir(opID), "OPERATION.md")
}

// OperationDir returns the directory for a classified operation.
func OperationDir(squad, opID string) string {
	return filepath.Join(root, "squads", squad, "operations", opID)
}

// OperationMDPath returns the OPERATION.md path for a classified operation.
func OperationMDPath(squad, opID string) string {
	return filepath.Join(OperationDir(squad, opID), "OPERATION.md")
}

// SchedulesDir returns the schedules directory.
func SchedulesDir() string { return filepath.Join(root, "schedules") }

// SquadsDir returns the runtime squads base directory.
func SquadsDir() string { return filepath.Join(root, "squads") }

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
