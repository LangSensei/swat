package runtime

import (
"encoding/json"
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"

"github.com/gofrs/flock"
)

// GeminiAdapter implements RuntimeAdapter for the Google Gemini CLI.
type GeminiAdapter struct {
BaseProvisioner
}

// NewGeminiAdapter creates a properly initialized GeminiAdapter.
func NewGeminiAdapter() *GeminiAdapter {
return &GeminiAdapter{
BaseProvisioner: BaseProvisioner{
dotDir:        ".gemini",
agentFileName: "GEMINI.md",
mcpConfigPath: ".gemini/settings.json",
},
}
}

// Name returns "gemini".
func (a *GeminiAdapter) Name() string { return "gemini" }

// ComposeHooks copies Gemini-specific hooks from each resolved skill.
// For each skill, it looks for <skillsRoot>/<skill>/hooks/gemini/.
// Script files (.js) are copied to <opDir>/.gemini/hooks/ preserving subdirectory structure.
// Each <skill>.json is read for its "hooks" object, and hook entries are appended
// to the corresponding event arrays in <opDir>/.gemini/settings.json.
func (a *GeminiAdapter) ComposeHooks(skillsRoot string, resolvedSkills []string, opDir string) error {
destHooksDir := filepath.Join(opDir, ".gemini", "hooks")
settingsPath := filepath.Join(opDir, ".gemini", "settings.json")

for _, skill := range resolvedSkills {
srcHooks := filepath.Join(skillsRoot, skill, "hooks", "gemini")
info, err := os.Stat(srcHooks)
if err != nil || !info.IsDir() {
continue
}

// Copy .js script files to .gemini/hooks/ preserving structure
if err := filepath.Walk(srcHooks, func(path string, fi os.FileInfo, err error) error {
if err != nil {
return err
}
if fi.IsDir() {
return nil
}
if !strings.HasSuffix(fi.Name(), ".js") {
return nil
}
rel, _ := filepath.Rel(srcHooks, path)
dest := filepath.Join(destHooksDir, rel)
if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
return err
}
data, readErr := os.ReadFile(path)
if readErr != nil {
return readErr
}
return os.WriteFile(dest, data, fi.Mode())
}); err != nil {
return fmt.Errorf("copy hook scripts for skill %s: %w", skill, err)
}

// Read <skill>.json for hook config and merge into settings.json
hookConfigPath := filepath.Join(srcHooks, skill+".json")
configData, err := os.ReadFile(hookConfigPath)
if err != nil {
continue // no config file for this skill, skip
}

var hookConfig map[string]interface{}
if err := json.Unmarshal(configData, &hookConfig); err != nil {
return fmt.Errorf("parse hooks config for skill %s: %w", skill, err)
}

incomingHooks, ok := hookConfig["hooks"].(map[string]interface{})
if !ok {
continue
}

// Read existing settings.json
if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
return fmt.Errorf("create .gemini dir: %w", err)
}

settings := make(map[string]interface{})
if data, err := os.ReadFile(settingsPath); err == nil {
if err := json.Unmarshal(data, &settings); err != nil {
return fmt.Errorf("parse settings.json: %w", err)
}
}

existingHooks, _ := settings["hooks"].(map[string]interface{})
if existingHooks == nil {
existingHooks = make(map[string]interface{})
}

// Append hook entries for each event key
for event, entries := range incomingHooks {
incomingArr, ok := entries.([]interface{})
if !ok {
continue
}
existingArr, _ := existingHooks[event].([]interface{})
existingArr = append(existingArr, incomingArr...)
existingHooks[event] = existingArr
}

settings["hooks"] = existingHooks

out, err := json.MarshalIndent(settings, "", "  ")
if err != nil {
return fmt.Errorf("marshal settings.json: %w", err)
}
out = append(out, '\n')
if err := os.WriteFile(settingsPath, out, 0644); err != nil {
return fmt.Errorf("write settings.json: %w", err)
}
}
return nil
}

// PrepareWorkspace registers the operation directory as a trusted folder in
// ~/.gemini/trustedFolders.json for all phases.
func (a *GeminiAdapter) PrepareWorkspace(opDir string, _ Phase) error {
homeDir, err := os.UserHomeDir()
if err != nil {
return fmt.Errorf("get home dir: %w", err)
}

geminiDir := filepath.Join(homeDir, ".gemini")
if err := os.MkdirAll(geminiDir, 0755); err != nil {
return fmt.Errorf("create ~/.gemini: %w", err)
}

trustedPath := filepath.Join(geminiDir, "trustedFolders.json")

fileLock := flock.New(trustedPath)
if err := fileLock.Lock(); err != nil {
return fmt.Errorf("acquire lock: %w", err)
}
defer fileLock.Unlock()

folders := make(map[string]string)
data, err := os.ReadFile(trustedPath)
if err == nil {
if err := json.Unmarshal(data, &folders); err != nil {
return fmt.Errorf("parse trustedFolders.json: %w", err)
}
}

key := strings.ReplaceAll(opDir, "\\", "/")
folders[key] = "TRUST_FOLDER"

out, err := json.MarshalIndent(folders, "", "  ")
if err != nil {
return fmt.Errorf("marshal trustedFolders.json: %w", err)
}
if err := os.WriteFile(trustedPath, out, 0644); err != nil {
return fmt.Errorf("write trustedFolders.json: %w", err)
}

return nil
}

// BuildCommand constructs the Gemini CLI command with standard flags.
func (a *GeminiAdapter) BuildCommand(prompt, workDir string) *exec.Cmd {
cmd := exec.Command("gemini", "-p", prompt, "--yolo", "--output-format", "json")
cmd.Dir = workDir
return cmd
}
