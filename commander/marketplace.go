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

// Browse lists all squads available in the marketplace
func (c *Commander) Browse() ([]BrowseResult, error) {
	// List squads/ directory via GitHub API
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
		// Fetch MANIFEST.md to get description
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

// Install fetches a squad from the marketplace and installs its dependencies
func (c *Commander) Install(squad string) error {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	squadDir := filepath.Join(bpDir, "squads", squad)

	if fileExists(filepath.Join(squadDir, "MANIFEST.md")) {
		return fmt.Errorf("squad %q is already installed", squad)
	}

	// Download squad from marketplace
	if err := ghDownloadDir("squads/"+squad, squadDir); err != nil {
		return fmt.Errorf("download squad %q: %w", squad, err)
	}

	// Verify MANIFEST.md exists
	if !fileExists(filepath.Join(squadDir, "MANIFEST.md")) {
		os.RemoveAll(squadDir)
		return fmt.Errorf("squad %q not found in marketplace", squad)
	}

	// Resolve all dependencies (framework + squad + transitive skills)
	allSkills := c.resolveDependencies(squad)
	allMCPs := c.resolveMCPDependencies(squad)

	// Install missing skills
	for _, skill := range allSkills {
		destSkill := filepath.Join(bpDir, "skills", skill)
		if fileExists(destSkill) {
			continue
		}
		if err := ghDownloadDir("skills/"+skill, destSkill); err != nil {
			return fmt.Errorf("download skill %q: %w", skill, err)
		}
	}

	// Install missing MCPs
	for _, mcp := range allMCPs {
		destMCP := filepath.Join(bpDir, "mcps", mcp+".json")
		if fileExists(destMCP) {
			continue
		}
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

// Update re-downloads a squad and its dependencies from the marketplace
func (c *Commander) Update(squad string) error {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	squadDir := filepath.Join(bpDir, "squads", squad)

	if !fileExists(filepath.Join(squadDir, "MANIFEST.md")) {
		return fmt.Errorf("squad %q is not installed", squad)
	}

	// Re-download squad (overwrite existing)
	if err := os.RemoveAll(squadDir); err != nil {
		return fmt.Errorf("remove old squad %q: %w", squad, err)
	}
	if err := ghDownloadDir("squads/"+squad, squadDir); err != nil {
		return fmt.Errorf("download squad %q: %w", squad, err)
	}

	// Resolve and update all dependencies
	allSkills := c.resolveDependencies(squad)
	allMCPs := c.resolveMCPDependencies(squad)

	for _, skill := range allSkills {
		destSkill := filepath.Join(bpDir, "skills", skill)
		os.RemoveAll(destSkill) // remove old version
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

	// Check for active operations
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

// ghListDir lists directory contents via GitHub Contents API
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

// ghGetFile downloads a single file's raw content
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

// ghDownloadDir recursively downloads a directory from the marketplace
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
			// Preserve executable bit for .sh files
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
