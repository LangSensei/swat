package layout

import (
	"os"
	"path/filepath"
)

var root string

// Init sets the swat root directory. Must be called once at startup.
func Init(swatRoot string) {
	if len(swatRoot) >= 2 && swatRoot[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			swatRoot = filepath.Join(home, swatRoot[2:])
		}
	}
	root = swatRoot
}

// Root returns the swat root directory.
func Root() string { return root }

// BlueprintsDir returns the blueprints base directory.
func BlueprintsDir() string { return filepath.Join(root, "blueprints") }

// SquadBlueprintDir returns the blueprint directory for a specific squad.
func SquadBlueprintDir(squad string) string {
	return filepath.Join(root, "blueprints", "squads", squad)
}

// FrameworkDir returns the _framework blueprint directory.
func FrameworkDir() string {
	return filepath.Join(root, "blueprints", "squads", "_framework")
}

// SkillsDir returns the blueprints/skills directory.
func SkillsDir() string { return filepath.Join(root, "blueprints", "skills") }

// MCPsDir returns the blueprints/mcps directory.
func MCPsDir() string { return filepath.Join(root, "blueprints", "mcps") }

// SchedulesDir returns the schedules directory.
func SchedulesDir() string { return filepath.Join(root, "schedules") }

// SquadsDir returns the runtime squads base directory.
func SquadsDir() string { return filepath.Join(root, "squads") }

// SquadDir returns the runtime directory for a squad.
func SquadDir(squad string) string { return filepath.Join(root, "squads", squad) }

// OperationDir returns the directory for a classified operation.
func OperationDir(squad, opID string) string {
	return filepath.Join(root, "squads", squad, "operations", opID)
}

// OperationMDPath returns the OPERATION.md path for a classified operation.
func OperationMDPath(squad, opID string) string {
	return filepath.Join(OperationDir(squad, opID), "OPERATION.md")
}

// UnclassifiedOperationDir returns the directory for an unclassified operation.
func UnclassifiedOperationDir(opID string) string {
	return filepath.Join(root, "squads", "_unclassified", "operations", opID)
}

// UnclassifiedOperationMDPath returns the OPERATION.md path for an unclassified operation.
func UnclassifiedOperationMDPath(opID string) string {
	return filepath.Join(UnclassifiedOperationDir(opID), "OPERATION.md")
}

// EnsureDirs creates the standard directory structure.
func EnsureDirs() {
	for _, dir := range []string{
		root,
		BlueprintsDir(),
		filepath.Join(BlueprintsDir(), "squads"),
		SkillsDir(),
		MCPsDir(),
		SquadsDir(),
		filepath.Join(SquadsDir(), "_unclassified", "operations"),
	} {
		os.MkdirAll(dir, 0755)
	}
}
