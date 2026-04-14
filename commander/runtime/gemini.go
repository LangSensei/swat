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

// PrepareWorkspace registers the operation directory as a trusted folder in
// ~/.gemini/trustedFolders.json for all phases. It reads the existing JSON
// object (or creates {} if not found), adds "<opDir>": "TRUST_FOLDER", and
// writes back. Paths use forward slashes.
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

	// Use forward slashes for the key
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
