package squads

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LangSensei/swat/commander/deps"
	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/platform"
)

// BrowseResult represents a squad available in the marketplace
type BrowseResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
}

// SkillPrereq represents a skill that has prerequisites needing user setup
type SkillPrereq struct {
	Skill string `json:"skill"`
	Path  string `json:"path"`
}

// List returns all installed squad blueprints.
func List() ([]map[string]string, error) {
	entries, err := os.ReadDir(layout.BlueprintSquadsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []map[string]string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_framework" {
			continue
		}
		info := map[string]string{"name": entry.Name()}
		manifestPath := filepath.Join(layout.BlueprintSquadDir(entry.Name()), "MANIFEST.md")
		if data, err := os.ReadFile(manifestPath); err == nil {
			if desc := deps.ExtractFrontmatterField(string(data), "description"); desc != "" {
				info["description"] = desc
			}
		}
		result = append(result, info)
	}
	return result, nil
}

// Count returns the number of installed squads.
func Count() int {
	squads, err := List()
	if err != nil {
		return 0
	}
	return len(squads)
}

// ListSummaries returns a human-readable summary of installed squads.
func ListSummaries() string {
	installed, err := List()
	if err != nil || len(installed) == 0 {
		return "(none installed)"
	}
	var lines []string
	for _, sq := range installed {
		desc := sq["description"]
		if desc == "" {
			desc = "(no description)"
		}
		lines = append(lines, fmt.Sprintf("• %s — %s", sq["name"], desc))
	}
	return strings.Join(lines, "\n")
}

// Browse lists all squads available in the marketplace.
func Browse() ([]BrowseResult, error) {
	entries, err := ghListDir("squads")
	if err != nil {
		return nil, fmt.Errorf("list marketplace squads: %w", err)
	}

	var results []BrowseResult
	for _, e := range entries {
		name := e.Name
		if e.Type != "dir" || name == "_framework" {
			continue
		}
		desc := ""
		data, err := ghGetFile("squads/" + name + "/MANIFEST.md")
		if err == nil {
			desc = deps.ExtractFrontmatterField(string(data), "description")
		}
		installed := platform.DirExists(layout.BlueprintSquadDir(name))
		results = append(results, BrowseResult{
			Name:        name,
			Description: desc,
			Installed:   installed,
		})
	}
	return results, nil
}

// collectPrereqs scans installed skills for prereq declarations in frontmatter.
func collectPrereqs(skills []string) []SkillPrereq {
	var prereqs []SkillPrereq
	for _, skill := range skills {
		skillMD := filepath.Join(layout.BlueprintSkillsDir(), skill, "SKILL.md")
		data, err := os.ReadFile(skillMD)
		if err != nil {
			continue
		}
		val := deps.ExtractFrontmatterField(string(data), "prereq")
		if val != "" {
			absPath := filepath.Join(layout.BlueprintSkillsDir(), skill, val)
			prereqs = append(prereqs, SkillPrereq{Skill: skill, Path: absPath})
		}
	}
	return prereqs
}

// Install fetches a squad from the marketplace and installs its dependencies.
func Install(squad string) ([]SkillPrereq, error) {
	squadDir := layout.BlueprintSquadDir(squad)

	if platform.FileExists(filepath.Join(squadDir, "MANIFEST.md")) {
		return nil, fmt.Errorf("squad %q is already installed", squad)
	}

	if err := ghDownloadDir("squads/"+squad, squadDir); err != nil {
		return nil, fmt.Errorf("download squad %q: %w", squad, err)
	}

	if !platform.FileExists(filepath.Join(squadDir, "MANIFEST.md")) {
		os.RemoveAll(squadDir)
		return nil, fmt.Errorf("squad %q not found in marketplace", squad)
	}

	allSkills := deps.ResolveSkillDependencies(squad)

	for _, skill := range allSkills {
		destSkill := filepath.Join(layout.BlueprintSkillsDir(), skill)
		if platform.FileExists(destSkill) {
			continue
		}
		if err := ghDownloadDir("skills/"+skill, destSkill); err != nil {
			return nil, fmt.Errorf("download skill %q: %w", skill, err)
		}
	}

	allMCPs := deps.ResolveMCPDependencies(squad, allSkills)

	for _, mcp := range allMCPs {
		destMCP := filepath.Join(layout.BlueprintMCPsDir(), mcp+".json")
		if platform.FileExists(destMCP) {
			continue
		}
		data, err := ghGetFile("mcps/" + mcp + ".json")
		if err != nil {
			return nil, fmt.Errorf("download MCP %q: %w", mcp, err)
		}
		os.MkdirAll(layout.BlueprintMCPsDir(), 0755)
		if err := os.WriteFile(destMCP, data, 0644); err != nil {
			return nil, fmt.Errorf("write MCP %q: %w", mcp, err)
		}
	}

	prereqs := collectPrereqs(allSkills)
	return prereqs, nil
}

// atomicDirReplace downloads remotePath into a temp directory, then atomically
// swaps it into destDir using rename. If the download fails, destDir is left
// unchanged. This eliminates the window where destDir is absent during updates.
func atomicDirReplace(remotePath, destDir string) error {
	tmpDir := destDir + ".tmp"
	oldDir := destDir + ".old"

	// Clean up any leftovers from a previously interrupted update.
	os.RemoveAll(tmpDir)
	os.RemoveAll(oldDir)

	// Download new version to temp directory.
	if err := ghDownloadDir(remotePath, tmpDir); err != nil {
		os.RemoveAll(tmpDir) // clean up on failure
		return err
	}

	// Swap: move current → .old, then move .tmp → current.
	// If destDir doesn't exist yet (first install of a skill during update),
	// skip the rename-aside step.
	if platform.DirExists(destDir) {
		if err := os.Rename(destDir, oldDir); err != nil {
			os.RemoveAll(tmpDir)
			return fmt.Errorf("rename old dir: %w", err)
		}
	}

	if err := os.Rename(tmpDir, destDir); err != nil {
		// Attempt to restore the old directory.
		os.Rename(oldDir, destDir)
		os.RemoveAll(tmpDir)
		return fmt.Errorf("rename new dir into place: %w", err)
	}

	// Clean up the old version (best-effort).
	os.RemoveAll(oldDir)
	return nil
}

// Update re-downloads a squad and its dependencies from the marketplace.
// It uses an atomic directory swap (download to temp, then rename) so the
// squad blueprint is never absent from disk during the update.
func Update(squad string) error {
	squadDir := layout.BlueprintSquadDir(squad)

	if !platform.FileExists(filepath.Join(squadDir, "MANIFEST.md")) {
		return fmt.Errorf("squad %q is not installed", squad)
	}

	if err := atomicDirReplace("squads/"+squad, squadDir); err != nil {
		return fmt.Errorf("update squad %q: %w", squad, err)
	}

	allSkills := deps.ResolveSkillDependencies(squad)

	for _, skill := range allSkills {
		destSkill := filepath.Join(layout.BlueprintSkillsDir(), skill)
		if err := atomicDirReplace("skills/"+skill, destSkill); err != nil {
			return fmt.Errorf("update skill %q: %w", skill, err)
		}
	}

	allMCPs := deps.ResolveMCPDependencies(squad, allSkills)

	for _, mcp := range allMCPs {
		destMCP := filepath.Join(layout.BlueprintMCPsDir(), mcp+".json")
		data, err := ghGetFile("mcps/" + mcp + ".json")
		if err != nil {
			return fmt.Errorf("download MCP %q: %w", mcp, err)
		}
		os.MkdirAll(layout.BlueprintMCPsDir(), 0755)
		if err := os.WriteFile(destMCP, data, 0644); err != nil {
			return fmt.Errorf("write MCP %q: %w", mcp, err)
		}
	}

	return nil
}

// Uninstall removes a squad blueprint and cleans up orphaned dependencies.
func Uninstall(squad string, purge bool) error {
	squadBP := layout.BlueprintSquadDir(squad)

	if !platform.FileExists(filepath.Join(squadBP, "MANIFEST.md")) {
		return fmt.Errorf("squad %q is not installed", squad)
	}

	if err := os.RemoveAll(squadBP); err != nil {
		return fmt.Errorf("remove squad blueprint: %w", err)
	}

	if purge {
		runtimeDir := layout.SquadDir(squad)
		if platform.DirExists(runtimeDir) {
			os.RemoveAll(runtimeDir)
		}
	}

	CleanOrphans()
	return nil
}

// CleanOrphans removes skills and MCPs not referenced by any installed squad.
func CleanOrphans() {
	installed, err := List()
	if err != nil {
		return
	}

	neededSkills := make(map[string]bool)
	neededMCPs := make(map[string]bool)

	for _, sq := range installed {
		name := sq["name"]
		skills := deps.ResolveSkillDependencies(name)
		for _, skill := range skills {
			neededSkills[skill] = true
		}
		for _, mcp := range deps.ResolveMCPDependencies(name, skills) {
			neededMCPs[mcp] = true
		}
	}

	skillsDir := layout.BlueprintSkillsDir()
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && !neededSkills[entry.Name()] {
				os.RemoveAll(filepath.Join(skillsDir, entry.Name()))
			}
		}
	}

	mcpsDir := layout.BlueprintMCPsDir()
	if entries, err := os.ReadDir(mcpsDir); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if filepath.Ext(name) == ".json" {
				base := name[:len(name)-5]
				if !neededMCPs[base] {
					os.Remove(filepath.Join(mcpsDir, name))
				}
			}
		}
	}
}
