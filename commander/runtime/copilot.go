package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"github.com/LangSensei/swat/commander/platform"
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

// ComposeHooks copies Copilot-specific hooks from each resolved skill.
// For each skill, it looks for <skillsRoot>/<skill>/hooks/copilot/ and copies
// the contents (scripts + JSON config files) to <opDir>/.github/hooks/.
func (a *CopilotAdapter) ComposeHooks(skillsRoot string, resolvedSkills []string, opDir string) error {
	destHooksDir := filepath.Join(opDir, ".github", "hooks")

	for _, skill := range resolvedSkills {
		srcHooks := filepath.Join(skillsRoot, skill, "hooks", "copilot")
		info, err := os.Stat(srcHooks)
		if err != nil || !info.IsDir() {
			continue
		}

		if err := platform.CopyDir(srcHooks, destHooksDir); err != nil {
			return fmt.Errorf("compose hooks for skill %s: %w", skill, err)
		}
	}
	return nil
}

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
