package runtime

import (
"encoding/json"
"fmt"
"os"
"path/filepath"
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

// ComposeMCPConfig merges incoming MCP configuration JSON into the existing
// config file at MCPConfigPath(). If the file doesn't exist, it is created.
// The incoming content's mcpServers are merged into the existing object so
// that multiple calls append servers rather than overwrite.
func (b *BaseProvisioner) ComposeMCPConfig(opDir string, content string) error {
dest := filepath.Join(opDir, b.mcpConfigPath)

// Ensure parent directory exists
if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
return fmt.Errorf("create dir for %s: %w", b.mcpConfigPath, err)
}

// Parse incoming config
var incoming map[string]interface{}
if err := json.Unmarshal([]byte(content), &incoming); err != nil {
return fmt.Errorf("parse incoming MCP config: %w", err)
}

// Read existing config (or start with empty object)
existing := make(map[string]interface{})
if data, err := os.ReadFile(dest); err == nil {
if err := json.Unmarshal(data, &existing); err != nil {
return fmt.Errorf("parse existing %s: %w", b.mcpConfigPath, err)
}
}

// Merge mcpServers
existingServers, _ := existing["mcpServers"].(map[string]interface{})
if existingServers == nil {
existingServers = make(map[string]interface{})
}
if incomingServers, ok := incoming["mcpServers"].(map[string]interface{}); ok {
for k, v := range incomingServers {
existingServers[k] = v
}
}
existing["mcpServers"] = existingServers

out, err := json.MarshalIndent(existing, "", "  ")
if err != nil {
return fmt.Errorf("marshal %s: %w", b.mcpConfigPath, err)
}
out = append(out, '\n')
if err := os.WriteFile(dest, out, 0644); err != nil {
return fmt.Errorf("write %s: %w", b.mcpConfigPath, err)
}
return nil
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
hooksExclude := map[string]bool{"hooks": true}

for _, skill := range resolvedSkills {
srcSkill := filepath.Join(skillsRoot, skill)
if _, err := os.Stat(srcSkill); err != nil {
continue
}

if err := copyDirExclude(srcSkill, filepath.Join(destSkillsDir, skill), hooksExclude); err != nil {
return fmt.Errorf("copy skill %s: %w", skill, err)
}
}
return nil
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
return copyDirExclude(src, dst, nil)
}

// copyDirExclude recursively copies a directory tree, skipping directories whose
// base name appears in the exclude set.
func copyDirExclude(src, dst string, exclude map[string]bool) error {
return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
if err != nil {
return err
}
rel, _ := filepath.Rel(src, path)
if info.IsDir() && exclude[info.Name()] && rel != "." {
return filepath.SkipDir
}
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
