// Package runtime defines the RuntimeAdapter interface for multi-runtime support.
// Each adapter encapsulates runtime-specific paths, file names, and command construction.
package runtime

import (
	"fmt"
	"os"
	"os/exec"
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

	// InstallHooks runs any runtime-specific initialization (e.g. git init for hook discovery)
	InstallHooks(opDir string) error

	// BuildCommand constructs the exec.Cmd for launching the runtime with the given prompt
	BuildCommand(prompt, workDir string) *exec.Cmd
}

// New creates a RuntimeAdapter by name. Reads the RUNTIME environment variable
// if name is empty, defaulting to "copilot".
func New(name string) (RuntimeAdapter, error) {
	if name == "" {
		name = os.Getenv("RUNTIME")
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
