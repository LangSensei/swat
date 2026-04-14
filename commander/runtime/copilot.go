package runtime

import (
	"os/exec"
)

// CopilotAdapter implements RuntimeAdapter for the GitHub Copilot CLI.
type CopilotAdapter struct {
	BaseProvisioner
}

// NewCopilotAdapter creates a properly initialized CopilotAdapter.
func NewCopilotAdapter() *CopilotAdapter {
	return &CopilotAdapter{
		BaseProvisioner: BaseProvisioner{
			dotDir:        ".github",
			agentFileName: "AGENTS.md",
			mcpConfigPath: ".mcp.json",
		},
	}
}

// Name returns "copilot".
func (a *CopilotAdapter) Name() string { return "copilot" }

// DotDir returns ".github".
func (a *CopilotAdapter) DotDir() string { return a.BaseProvisioner.dotDir }

// AgentFileName returns "AGENTS.md".
func (a *CopilotAdapter) AgentFileName() string { return a.BaseProvisioner.agentFileName }

// MCPConfigPath returns ".mcp.json".
func (a *CopilotAdapter) MCPConfigPath() string { return a.BaseProvisioner.mcpConfigPath }

// InstallHooks runs git init so that Copilot CLI can discover .github/hooks/.
func (a *CopilotAdapter) InstallHooks(opDir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = opDir
	return cmd.Run() // best-effort; hooks won't fire without it but operation can proceed
}

// BuildCommand constructs the Copilot CLI command with standard flags.
func (a *CopilotAdapter) BuildCommand(prompt, workDir string) *exec.Cmd {
	cmd := exec.Command("copilot", "-p", prompt, "--yolo", "--output-format", "json", "--effort", "xhigh")
	cmd.Dir = workDir
	return cmd
}
