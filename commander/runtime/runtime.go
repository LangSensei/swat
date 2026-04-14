// Package runtime defines the RuntimeAdapter interface for multi-runtime support.
// Each adapter encapsulates runtime-specific paths, file names, and command construction.
package runtime

import (
"fmt"
"os/exec"
"strings"
)

// Phase indicates which stage of the dispatch pipeline is calling PrepareWorkspace.
type Phase string

const (
// PhaseClassify is the classify+enrich stage (lightweight, no git init needed).
PhaseClassify Phase = "classify"
// PhaseOperate is the provision+launch stage (full workspace setup).
PhaseOperate Phase = "operate"
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

// ComposeAgentFile writes the agent instruction file into the operation directory
ComposeAgentFile(opDir string, content []byte) error

// ComposeMCPConfig merges incoming MCP configuration JSON into the existing
// config file (or creates it). Multiple calls append mcpServers rather than overwrite.
ComposeMCPConfig(opDir string, content string) error

// ComposeSquad copies the squad blueprint snapshot into opDir/.squad/
ComposeSquad(squadBPDir, opDir string) error

// ComposeSkills copies resolved skill content (excluding hooks/) into the runtime's dotDir
ComposeSkills(skillsRoot string, resolvedSkills []string, opDir string) error

// ComposeHooks copies runtime-specific hooks from each resolved skill's
// hooks/<runtime>/ subdirectory into the operation directory
ComposeHooks(skillsRoot string, resolvedSkills []string, opDir string) error

// PrepareWorkspace runs runtime-specific workspace initialization for the given phase.
// During PhaseClassify this is a no-op for most runtimes; during PhaseOperate it may
// run git init or other setup needed for hook discovery.
PrepareWorkspace(opDir string, phase Phase) error

// BuildCommand constructs the exec.Cmd for launching the runtime with the given prompt
BuildCommand(prompt, workDir string) *exec.Cmd
}

// New creates a RuntimeAdapter for the given runtime name.
// If name is empty, defaults to "copilot".
func New(name string) (RuntimeAdapter, error) {
if name == "" {
name = "copilot"
}
name = strings.ToLower(name)

switch name {
case "copilot":
return NewCopilotAdapter(), nil
case "gemini":
return NewGeminiAdapter(), nil
default:
return nil, fmt.Errorf("unknown runtime: %q", name)
}
}
