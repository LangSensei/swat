// Package runtime defines the RuntimeAdapter interface for multi-runtime support.
// Each adapter encapsulates runtime-specific paths, file names, and command construction.
package runtime

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RuntimeAdapter abstracts the agent runtime (Copilot CLI, Claude Code, etc.)
type RuntimeAdapter interface {
	// Name returns the runtime identifier (e.g. "copilot")
	Name() string

	// DotDir returns the dotfile directory name used by the runtime (e.g. ".github")
	DotDir() string

	// AgentFileName returns the file the runtime reads for agent instructions (e.g. "AGENTS.md")
	AgentFileName() string

	// MCPConfigPath returns the MCP configuration file path relative to operation root (e.g. ".mcp.json")
	MCPConfigPath() string

	// WriteAgentFile writes the agent instruction file into the operation directory
	WriteAgentFile(opDir string, content []byte) error

	// WriteMCPConfig writes the MCP configuration into the operation directory
	WriteMCPConfig(opDir string, content string) error

	// CopySquad copies the squad blueprint snapshot into opDir/.squad/
	CopySquad(squadBPDir, opDir string) error

	// CopySkills copies resolved skills into the runtime's dotDir (skills + hooks)
	CopySkills(skillsRoot string, resolvedSkills []string, opDir string) error

	// InstallHooks runs any runtime-specific initialization (e.g. git init for hook discovery)
	InstallHooks(opDir string) error

	// BuildCommand constructs the exec.Cmd for launching the runtime with the given prompt
	BuildCommand(prompt, workDir string) *exec.Cmd
}

// New creates a RuntimeAdapter by name. Reads the RUNTIME setting from
// ~/.swat/swat.env if name is empty, defaulting to "copilot" if the file
// is missing or does not contain a RUNTIME line. This allows dynamic runtime
// switching between operations without restarting Commander.
func New(name string) (RuntimeAdapter, error) {
	if name == "" {
		name = readRuntimeFromEnvFile()
	}
	if name == "" {
		name = "copilot"
	}
	name = strings.ToLower(name)

	switch name {
	case "copilot":
		return NewCopilotAdapter(), nil
	default:
		return nil, fmt.Errorf("unknown runtime: %q", name)
	}
}

// readRuntimeFromEnvFile reads the RUNTIME value from ~/.swat/swat.env.
// Returns empty string if the file doesn't exist or has no RUNTIME line.
func readRuntimeFromEnvFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	f, err := os.Open(filepath.Join(home, ".swat", "swat.env"))
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		if strings.TrimSpace(key) == "RUNTIME" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
