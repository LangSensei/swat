package commander

import "path/filepath"

// Layout encapsulates the ~/.swat directory structure.
type Layout struct {
	Root string // e.g. ~/.swat
}

// BlueprintsDir returns the blueprints base directory.
func (l *Layout) BlueprintsDir() string {
	return filepath.Join(l.Root, "blueprints")
}

// SquadBlueprintDir returns the blueprint directory for a specific squad.
func (l *Layout) SquadBlueprintDir(squad string) string {
	return filepath.Join(l.Root, "blueprints", "squads", squad)
}

// FrameworkDir returns the _framework blueprint directory.
func (l *Layout) FrameworkDir() string {
	return filepath.Join(l.Root, "blueprints", "squads", "_framework")
}

// SkillsDir returns the blueprints/skills directory.
func (l *Layout) SkillsDir() string {
	return filepath.Join(l.Root, "blueprints", "skills")
}

// MCPsDir returns the blueprints/mcps directory.
func (l *Layout) MCPsDir() string {
	return filepath.Join(l.Root, "blueprints", "mcps")
}

// SchedulesDir returns the schedules directory.
func (l *Layout) SchedulesDir() string {
	return filepath.Join(l.Root, "schedules")
}

// SquadsDir returns the runtime squads base directory.
func (l *Layout) SquadsDir() string {
	return filepath.Join(l.Root, "squads")
}

// SquadDir returns the runtime directory for a squad (squads/{squad}).
func (l *Layout) SquadDir(squad string) string {
	return filepath.Join(l.Root, "squads", squad)
}

// OperationDir returns the directory for a classified operation.
func (l *Layout) OperationDir(squad, opID string) string {
	return filepath.Join(l.Root, "squads", squad, "operations", opID)
}

// OperationMDPath returns the OPERATION.md path for a classified operation.
func (l *Layout) OperationMDPath(squad, opID string) string {
	return filepath.Join(l.OperationDir(squad, opID), "OPERATION.md")
}

// UnclassifiedOperationDir returns the directory for an unclassified operation.
func (l *Layout) UnclassifiedOperationDir(opID string) string {
	return filepath.Join(l.Root, "squads", "_unclassified", "operations", opID)
}

// UnclassifiedOperationMDPath returns the OPERATION.md path for an unclassified operation.
func (l *Layout) UnclassifiedOperationMDPath(opID string) string {
	return filepath.Join(l.UnclassifiedOperationDir(opID), "OPERATION.md")
}
