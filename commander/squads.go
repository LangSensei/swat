package commander

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const marketplaceAPI = "https://api.github.com/repos/LangSensei/swat-marketplace"

// BrowseResult represents a squad available in the marketplace
type BrowseResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
}

// ListSquads returns all installed squad blueprints
func (c *Commander) ListSquads() ([]map[string]string, error) {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	entries, err := os.ReadDir(filepath.Join(bpDir, "squads"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var squads []map[string]string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_framework" {
			continue
		}
		info := map[string]string{"name": entry.Name()}
		manifestPath := filepath.Join(bpDir, "squads", entry.Name(), "MANIFEST.md")
		if data, err := os.ReadFile(manifestPath); err == nil {
			if desc := extractFrontmatterField(string(data), "description"); desc != "" {
				info["description"] = desc
			}
		}
		squads = append(squads, info)
	}
	return squads, nil
}

// countSquads returns the number of installed squads
func (c *Commander) countSquads() int {
	squads, err := c.ListSquads()
	if err != nil {
		return 0
	}
	return len(squads)
}

// listSquadSummaries returns a formatted string of installed squads and their descriptions
func (c *Commander) listSquadSummaries() string {
	squads, err := c.ListSquads()
	if err != nil || len(squads) == 0 {
		return "(none installed)"
	}
	var lines []string
	for _, sq := range squads {
		desc := sq["description"]
		if desc == "" {
			desc = "(no description)"
		}
		lines = append(lines, fmt.Sprintf("• %s — %s", sq["name"], desc))
	}
	return strings.Join(lines, "\n")
}

// Browse lists all squads available in the marketplace
func (c *Commander) Browse() ([]BrowseResult, error) {
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
			desc = extractFrontmatterField(string(data), "description")
		}
		installed := dirExists(filepath.Join(c.SwatRoot, "blueprints", "squads", name))
		results = append(results, BrowseResult{
			Name:        name,
			Description: desc,
			Installed:   installed,
		})
	}
	return results, nil
}

// SkillPrereq represents a skill that has prerequisites needing user setup
type SkillPrereq struct {
	Skill string `json:"skill"`
	Path  string `json:"path"` // relative path within the skill dir, e.g. "references/setup.md"
}

// collectPrereqs scans installed skills for prereq declarations in frontmatter
func (c *Commander) collectPrereqs(skills []string) []SkillPrereq {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	var prereqs []SkillPrereq
	for _, skill := range skills {
		skillMD := filepath.Join(bpDir, "skills", skill, "SKILL.md")
		data, err := os.ReadFile(skillMD)
		if err != nil {
			continue
		}
		val := parseFrontmatterValue(string(data), "prereq")
		if val != "" {
			absPath := filepath.Join(bpDir, "skills", skill, val)
			prereqs = append(prereqs, SkillPrereq{Skill: skill, Path: absPath})
		}
	}
	return prereqs
}

// parseFrontmatterValue extracts a single string value from frontmatter
func parseFrontmatterValue(md, field string) string {
	if !strings.HasPrefix(md, "---") {
		return ""
	}
	end := strings.Index(md[3:], "\n---")
	if end < 0 {
		return ""
	}
	fm := md[4 : end+3]
	for _, line := range strings.Split(fm, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, field+":") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, field+":"))
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return ""
}

// Install fetches a squad from the marketplace and installs its dependencies.
// Returns a list of skill prereqs that need user attention (may be empty).
func (c *Commander) Install(squad string) ([]SkillPrereq, error) {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	squadDir := filepath.Join(bpDir, "squads", squad)

	if fileExists(filepath.Join(squadDir, "MANIFEST.md")) {
		return nil, fmt.Errorf("squad %q is already installed", squad)
	}

	if err := ghDownloadDir("squads/"+squad, squadDir); err != nil {
		return nil, fmt.Errorf("download squad %q: %w", squad, err)
	}

	if !fileExists(filepath.Join(squadDir, "MANIFEST.md")) {
		os.RemoveAll(squadDir)
		return nil, fmt.Errorf("squad %q not found in marketplace", squad)
	}

	allSkills := c.resolveDependencies(squad)
	allMCPs := c.resolveMCPDependencies(squad)

	for _, skill := range allSkills {
		destSkill := filepath.Join(bpDir, "skills", skill)
		if fileExists(destSkill) {
			continue
		}
		if err := ghDownloadDir("skills/"+skill, destSkill); err != nil {
			return nil, fmt.Errorf("download skill %q: %w", skill, err)
		}
	}

	for _, mcp := range allMCPs {
		destMCP := filepath.Join(bpDir, "mcps", mcp+".json")
		if fileExists(destMCP) {
			continue
		}
		data, err := ghGetFile("mcps/" + mcp + ".json")
		if err != nil {
			return nil, fmt.Errorf("download MCP %q: %w", mcp, err)
		}
		os.MkdirAll(filepath.Join(bpDir, "mcps"), 0755)
		if err := os.WriteFile(destMCP, data, 0644); err != nil {
			return nil, fmt.Errorf("write MCP %q: %w", mcp, err)
		}
	}

	prereqs := c.collectPrereqs(allSkills)
	return prereqs, nil
}

// Update re-downloads a squad and its dependencies from the marketplace
func (c *Commander) Update(squad string) error {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	squadDir := filepath.Join(bpDir, "squads", squad)

	if !fileExists(filepath.Join(squadDir, "MANIFEST.md")) {
		return fmt.Errorf("squad %q is not installed", squad)
	}

	if err := os.RemoveAll(squadDir); err != nil {
		return fmt.Errorf("remove old squad %q: %w", squad, err)
	}
	if err := ghDownloadDir("squads/"+squad, squadDir); err != nil {
		return fmt.Errorf("download squad %q: %w", squad, err)
	}

	allSkills := c.resolveDependencies(squad)
	allMCPs := c.resolveMCPDependencies(squad)

	for _, skill := range allSkills {
		destSkill := filepath.Join(bpDir, "skills", skill)
		os.RemoveAll(destSkill)
		if err := ghDownloadDir("skills/"+skill, destSkill); err != nil {
			return fmt.Errorf("download skill %q: %w", skill, err)
		}
	}

	for _, mcp := range allMCPs {
		destMCP := filepath.Join(bpDir, "mcps", mcp+".json")
		data, err := ghGetFile("mcps/" + mcp + ".json")
		if err != nil {
			return fmt.Errorf("download MCP %q: %w", mcp, err)
		}
		os.MkdirAll(filepath.Join(bpDir, "mcps"), 0755)
		if err := os.WriteFile(destMCP, data, 0644); err != nil {
			return fmt.Errorf("write MCP %q: %w", mcp, err)
		}
	}

	return nil
}

// Uninstall removes a squad blueprint and cleans up orphaned dependencies
func (c *Commander) Uninstall(squad string, purge bool) error {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	squadBP := filepath.Join(bpDir, "squads", squad)

	if !fileExists(filepath.Join(squadBP, "MANIFEST.md")) {
		return fmt.Errorf("squad %q is not installed", squad)
	}

	ops, err := c.ListOperations()
	if err == nil {
		for _, op := range ops {
			if op.Squad == squad && op.Status == "active" {
				return fmt.Errorf("squad %q has active operation %s — cancel it first", squad, op.OperationID)
			}
		}
	}

	if err := os.RemoveAll(squadBP); err != nil {
		return fmt.Errorf("remove squad blueprint: %w", err)
	}

	if purge {
		runtimeDir := c.SquadDir(squad)
		if fileExists(runtimeDir) {
			os.RemoveAll(runtimeDir)
		}
	}

	c.cleanOrphans()
	return nil
}

// cleanOrphans removes skills and MCPs not referenced by any installed squad
func (c *Commander) cleanOrphans() {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")

	neededSkills := make(map[string]bool)
	neededMCPs := make(map[string]bool)

	squads, err := c.ListSquads()
	if err != nil {
		return
	}

	for _, sq := range squads {
		name := sq["name"]
		for _, skill := range c.resolveDependencies(name) {
			neededSkills[skill] = true
		}
		for _, mcp := range c.resolveMCPDependencies(name) {
			neededMCPs[mcp] = true
		}
	}

	skillsDir := filepath.Join(bpDir, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && !neededSkills[entry.Name()] {
				os.RemoveAll(filepath.Join(skillsDir, entry.Name()))
			}
		}
	}

	mcpsDir := filepath.Join(bpDir, "mcps")
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

// --- GitHub API helpers ---

type ghEntry struct {
	Name string `json:"name"`
	Type string `json:"type"` // "file" or "dir"
	Path string `json:"path"`
}

func ghListDir(path string) ([]ghEntry, error) {
	url := fmt.Sprintf("%s/contents/%s", marketplaceAPI, path)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("path %q not found in marketplace", path)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var entries []ghEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func ghGetFile(path string) ([]byte, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/LangSensei/swat-marketplace/main/%s", path)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("file %q not found", path)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, path)
	}

	return io.ReadAll(resp.Body)
}

func ghDownloadDir(remotePath, localDir string) error {
	entries, err := ghListDir(remotePath)
	if err != nil {
		return err
	}

	os.MkdirAll(localDir, 0755)

	for _, e := range entries {
		localPath := filepath.Join(localDir, e.Name)
		switch e.Type {
		case "file":
			data, err := ghGetFile(e.Path)
			if err != nil {
				return fmt.Errorf("download %s: %w", e.Path, err)
			}
			perm := os.FileMode(0644)
			if strings.HasSuffix(e.Name, ".sh") {
				perm = 0755
			}
			if err := os.WriteFile(localPath, data, perm); err != nil {
				return err
			}
		case "dir":
			if err := ghDownloadDir(e.Path, localPath); err != nil {
				return err
			}
		}
	}
	return nil
}
