package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LangSensei/swat/commander/layout"
)

// BaseProvisioner provides shared provisioning logic used across all runtime adapters.
type BaseProvisioner struct {
	dotDir        string
	agentFileName string
	mcpConfigPath string
}

// DotDir returns the runtime-specific dot directory (e.g. ".github").
func (b *BaseProvisioner) DotDir() string { return b.dotDir }

// AgentFileName returns the runtime-specific agent file name (e.g. "AGENTS.md").
func (b *BaseProvisioner) AgentFileName() string { return b.agentFileName }

// MCPConfigPath returns the runtime-specific MCP config path (e.g. ".mcp.json").
func (b *BaseProvisioner) MCPConfigPath() string { return b.mcpConfigPath }

// ComposeAgentFile writes the agent instruction content to the appropriate file in opDir.
func (b *BaseProvisioner) ComposeAgentFile(opDir string, content []byte) error {
	dest := filepath.Join(opDir, b.agentFileName)
	if err := os.WriteFile(dest, content, 0644); err != nil {
		return fmt.Errorf("write %s: %w", b.agentFileName, err)
	}
	return nil
}

// ComposeMCPConfig reads MCP blueprint JSON files, injects runtime/notify args
// into the swat server entry, and writes the combined config to the operation dir.
func (b *BaseProvisioner) ComposeMCPConfig(opDir, runtimeName, notify string, mcps []string) error {
	servers := make(map[string]interface{})
	mcpsDir := layout.BlueprintMCPsDir()

	for _, name := range mcps {
		path := filepath.Join(mcpsDir, name+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		raw := strings.TrimSpace(string(data))

		// Inject --runtime and --notify into the swat server entry
		if name == "swat" {
			raw = injectSwatArgs(raw, runtimeName, notify)
		}

		var parsed interface{}
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			continue
		}
		servers[name] = parsed
	}

	if len(servers) == 0 {
		return nil
	}

	dest := filepath.Join(opDir, b.mcpConfigPath)
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("create dir for %s: %w", b.mcpConfigPath, err)
	}

	existing := make(map[string]interface{})
	if data, err := os.ReadFile(dest); err == nil {
		json.Unmarshal(data, &existing)
	}
	existing["mcpServers"] = servers

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", b.mcpConfigPath, err)
	}
	out = append(out, '\n')
	return os.WriteFile(dest, out, 0644)
}

// injectSwatArgs adds --runtime and --notify flags to a swat MCP server JSON config.
func injectSwatArgs(raw, runtimeName, notify string) string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return raw
	}
	var existing []string
	if args, ok := obj["args"]; ok {
		if arr, ok := args.([]interface{}); ok {
			for _, a := range arr {
				if s, ok := a.(string); ok {
					existing = append(existing, s)
				}
			}
		}
	}
	if runtimeName != "" {
		existing = append(existing, "--runtime", runtimeName)
	}
	if notify != "" {
		existing = append(existing, "--notify", notify)
	}
	obj["args"] = existing
	out, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return string(out)
}

// ComposeSquad copies the squad blueprint snapshot into opDir/.squad/.
func (b *BaseProvisioner) ComposeSquad(squadBPDir, opDir string) error {
	destDir := filepath.Join(opDir, ".squad")
	if err := copyDir(squadBPDir, destDir); err != nil {
		return fmt.Errorf("copy squad snapshot: %w", err)
	}
	return nil
}

// ComposeSkills copies resolved skill content into the runtime's dotDir.
// Only skill content (SKILL.md, etc.) is copied to <dotDir>/skills/<name>/.
// The hooks/ subdirectory is excluded entirely — hooks are handled by ComposeHooks.
func (b *BaseProvisioner) ComposeSkills(skillsRoot string, resolvedSkills []string, opDir string) error {
	destSkillsDir := filepath.Join(opDir, b.dotDir, "skills")

	for _, skill := range resolvedSkills {
		srcSkill := filepath.Join(skillsRoot, skill)
		if _, err := os.Stat(srcSkill); err != nil {
			continue
		}

		dest := filepath.Join(destSkillsDir, skill)
		if err := filepath.Walk(srcSkill, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(srcSkill, path)
			if info.IsDir() && info.Name() == "hooks" && rel != "." {
				return filepath.SkipDir
			}
			target := filepath.Join(dest, rel)
			if info.IsDir() {
				return os.MkdirAll(target, 0755)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, info.Mode())
		}); err != nil {
			return fmt.Errorf("copy skill %s: %w", skill, err)
		}
	}
	return nil
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
