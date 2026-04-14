package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gofrs/flock"
	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/platform"
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
// For each skill, it copies the entire hooks/gemini/ directory to <opDir>/.gemini/hooks/
// and collects hook JSON configs. After all skills are processed, it reads settings.json
// once, merges all accumulated hooks, and writes it back once.
func (a *GeminiAdapter) ComposeHooks(skillsRoot string, resolvedSkills []string, opDir string) error {
	destHooksDir := filepath.Join(opDir, ".gemini", "hooks")
	settingsPath := filepath.Join(opDir, ".gemini", "settings.json")

	// Accumulate all hook entries across skills
	allHooks := make(map[string][]interface{})

	for _, skill := range resolvedSkills {
		srcHooks := filepath.Join(skillsRoot, skill, "hooks", "gemini")
		info, err := os.Stat(srcHooks)
		if err != nil || !info.IsDir() {
			continue
		}

		// Copy entire hooks/gemini/ directory to .gemini/hooks/
		if err := platform.CopyDir(srcHooks, destHooksDir); err != nil {
			return fmt.Errorf("copy hooks for skill %s: %w", skill, err)
		}

		// Collect hook JSON configs from root-level *.json files
		entries, err := os.ReadDir(srcHooks)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			configData, err := os.ReadFile(filepath.Join(srcHooks, entry.Name()))
			if err != nil {
				continue
			}
			var hookConfig map[string]interface{}
			if err := json.Unmarshal(configData, &hookConfig); err != nil {
				return fmt.Errorf("parse hooks config %s for skill %s: %w", entry.Name(), skill, err)
			}
			incomingHooks, ok := hookConfig["hooks"].(map[string]interface{})
			if !ok {
				continue
			}
			for event, entries := range incomingHooks {
				incomingArr, ok := entries.([]interface{})
				if !ok {
					continue
				}
				allHooks[event] = append(allHooks[event], incomingArr...)
			}
		}
	}

	// Batch write: merge all accumulated hooks into settings.json once
	if len(allHooks) > 0 {
		if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
			return fmt.Errorf("create .gemini dir: %w", err)
		}

		settings := make(map[string]interface{})
		if data, err := os.ReadFile(settingsPath); err == nil {
			if err := json.Unmarshal(data, &settings); err != nil {
				return fmt.Errorf("parse settings.json: %w", err)
			}
		}

		// Set hooks (composed once per provisioning, no merge needed)
		settings["hooks"] = allHooks

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

	fileLock := flock.New(trustedPath + ".lock")
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

	// Trust the entire .swat directory so Gemini CLI can read blueprints
	swatRoot := strings.ReplaceAll(layout.Root(), "\\", "/")
	folders[swatRoot] = "TRUST_FOLDER"

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
