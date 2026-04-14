package deps

import (
	"os"
	"path/filepath"

	"github.com/LangSensei/swat/commander/layout"
)

// ResolveSkillDependencies recursively resolves all skill dependencies for a squad.
func ResolveSkillDependencies(squad string) []string {
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

// ResolveMCPDependencies collects all MCP names from protocol, manifest, and resolved skills.
func ResolveMCPDependencies(squad string, resolvedSkills []string) []string {
	seen := make(map[string]bool)
	var result []string

	collect := func(data []byte) {
		for _, m := range ParseDependencyList(string(data), "mcps") {
			if !seen[m] {
				seen[m] = true
				result = append(result, m)
			}
		}
	}

	if data, err := os.ReadFile(filepath.Join(layout.BlueprintFrameworkDir(), "PROTOCOL.md")); err == nil {
		collect(data)
	}
	if data, err := os.ReadFile(filepath.Join(layout.BlueprintSquadDir(squad), "MANIFEST.md")); err == nil {
		collect(data)
	}
	for _, skill := range resolvedSkills {
		skillMD := filepath.Join(layout.BlueprintSkillsDir(), skill, "SKILL.md")
		if data, err := os.ReadFile(skillMD); err == nil {
			collect(data)
		}
	}
	return result
}
