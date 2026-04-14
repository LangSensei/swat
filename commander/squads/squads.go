package squads

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/LangSensei/swat/commander/deps"
	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/platform"
)

const marketplaceAPI = "https://api.github.com/repos/LangSensei/swat-marketplace"

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
	entries, err := os.ReadDir(layout.BlueprintSquadsDir())
	if err != nil {
		return "(none installed)"
	}
	var lines []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_framework" {
			continue
		}
		name := entry.Name()
		desc := "(no description)"
		manifestPath := filepath.Join(layout.BlueprintSquadDir(name), "MANIFEST.md")
		if data, err := os.ReadFile(manifestPath); err == nil {
			if d := deps.ExtractFrontmatterField(string(data), "description"); d != "" {
				desc = d
			}
		}
		lines = append(lines, fmt.Sprintf("• %s — %s", name, desc))
	}
	if len(lines) == 0 {
		return "(none installed)"
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
		val := deps.ParseFrontmatterValue(string(data), "prereq")
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

// Update re-downloads a squad and its dependencies from the marketplace.
func Update(squad string) error {
	squadDir := layout.BlueprintSquadDir(squad)

	if !platform.FileExists(filepath.Join(squadDir, "MANIFEST.md")) {
		return fmt.Errorf("squad %q is not installed", squad)
	}

	if err := os.RemoveAll(squadDir); err != nil {
		return fmt.Errorf("remove old squad %q: %w", squad, err)
	}
	if err := ghDownloadDir("squads/"+squad, squadDir); err != nil {
		return fmt.Errorf("download squad %q: %w", squad, err)
	}

	allSkills := deps.ResolveSkillDependencies(squad)

	for _, skill := range allSkills {
		destSkill := filepath.Join(layout.BlueprintSkillsDir(), skill)
		os.RemoveAll(destSkill)
		if err := ghDownloadDir("skills/"+skill, destSkill); err != nil {
			return fmt.Errorf("download skill %q: %w", skill, err)
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
// activeOps is a list of operation IDs currently active for this squad (caller checks).
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
		if platform.FileExists(runtimeDir) {
			os.RemoveAll(runtimeDir)
		}
	}

	CleanOrphans()
	return nil
}

// CleanOrphans removes skills and MCPs not referenced by any installed squad.
func CleanOrphans() {
	neededSkills := make(map[string]bool)
	neededMCPs := make(map[string]bool)

	installed, err := List()
	if err != nil {
		return
	}

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

// --- GitHub API helpers ---

var (
	ghTokenOnce  sync.Once
	ghTokenValue string
)

func resolveGHToken() string {
	ghTokenOnce.Do(func() {
		if t := os.Getenv("GITHUB_TOKEN"); t != "" {
			ghTokenValue = t
			return
		}
		if out, err := exec.Command("gh", "auth", "token").Output(); err == nil {
			ghTokenValue = strings.TrimSpace(string(out))
		}
	})
	return ghTokenValue
}

func ghHTTPGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if token := resolveGHToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return http.DefaultClient.Do(req)
}

type ghEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

func ghListDir(path string) ([]ghEntry, error) {
	url := fmt.Sprintf("%s/contents/%s", marketplaceAPI, path)
	resp, err := ghHTTPGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("path %q not found in marketplace", path)
	}
	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("GitHub API rate limited (403) — install gh CLI or set GITHUB_TOKEN")
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
	resp, err := ghHTTPGet(url)
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
