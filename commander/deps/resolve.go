package deps

import (
	"os"
	"path/filepath"

	"github.com/LangSensei/swat/commander/layout"
)

// ResolveDependencies recursively resolves all skill dependencies for a squad.
func ResolveDependencies(squad string) []string {
	visited := make(map[string]bool)
	var result []string

	var seeds []string
	protocolPath := filepath.Join(layout.BlueprintFrameworkDir(), "PROTOCOL.md")
	if data, err := os.ReadFile(protocolPath); err == nil {
		seeds = append(seeds, ParseDependencyList(string(data), "skills")...)
	}
	manifestPath := filepath.Join(layout.BlueprintSquadDir(squad), "MANIFEST.md")
	if data, err := os.ReadFile(manifestPath); err == nil {
		seeds = append(seeds, ParseDependencyList(string(data), "skills")...)
	}

	queue := seeds
	for len(queue) > 0 {
		skill := queue[0]
		queue = queue[1:]
		if visited[skill] {
			continue
		}
		visited[skill] = true
		result = append(result, skill)

		skillMD := filepath.Join(layout.BlueprintSkillsDir(), skill, "SKILL.md")
		if data, err := os.ReadFile(skillMD); err == nil {
			for _, dep := range ParseDependencyList(string(data), "skills") {
				if !visited[dep] {
					queue = append(queue, dep)
				}
			}
		}
	}
	return result
}

// ResolveMCPDependencies collects all MCP names from protocol, manifest, and transitive skills.
func ResolveMCPDependencies(squad string) []string {
	seen := make(map[string]bool)
	var result []string

	if data, err := os.ReadFile(filepath.Join(layout.BlueprintFrameworkDir(), "PROTOCOL.md")); err == nil {
		for _, m := range ParseDependencyList(string(data), "mcps") {
			if !seen[m] {
				seen[m] = true
				result = append(result, m)
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(layout.BlueprintSquadDir(squad), "MANIFEST.md")); err == nil {
		for _, m := range ParseDependencyList(string(data), "mcps") {
			if !seen[m] {
				seen[m] = true
				result = append(result, m)
			}
		}
	}
	for _, skill := range ResolveDependencies(squad) {
		skillMD := filepath.Join(layout.BlueprintSkillsDir(), skill, "SKILL.md")
		if data, err := os.ReadFile(skillMD); err == nil {
			for _, m := range ParseDependencyList(string(data), "mcps") {
				if !seen[m] {
					seen[m] = true
					result = append(result, m)
				}
			}
		}
	}
	return result
}
