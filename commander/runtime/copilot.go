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

// PrepareWorkspace runs workspace initialization for the given phase.
// During PhaseOperate it runs git init so that Copilot CLI can discover .github/hooks/.
// During PhaseClassify no initialization is needed.
func (a *CopilotAdapter) PrepareWorkspace(opDir string, phase Phase) error {
	if phase != PhaseOperate {
		return nil
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = opDir
	return cmd.Run()
}

// BuildCommand constructs the Copilot CLI command with standard flags.
func (a *CopilotAdapter) BuildCommand(prompt, workDir string) *exec.Cmd {
	cmd := exec.Command("copilot", "-p", prompt, "--yolo", "--output-format", "json", "--effort", "xhigh")
	cmd.Dir = workDir
	return cmd
}
