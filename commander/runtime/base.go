package runtime

import (
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

// WriteAgentFile writes the agent instruction content to the appropriate file in opDir.
func (b *BaseProvisioner) WriteAgentFile(opDir string, content []byte) error {
	dest := filepath.Join(opDir, b.agentFileName)
	if err := os.WriteFile(dest, content, 0644); err != nil {
		return fmt.Errorf("write %s: %w", b.agentFileName, err)
	}
	return nil
}

// WriteMCPConfig writes the MCP configuration JSON to the appropriate path in opDir.
func (b *BaseProvisioner) WriteMCPConfig(opDir string, content string) error {
	dest := filepath.Join(opDir, b.mcpConfigPath)
	if err := os.WriteFile(dest, []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", b.mcpConfigPath, err)
	}
	return nil
}

// CopySquad copies the squad blueprint snapshot into opDir/.squad/.
func (b *BaseProvisioner) CopySquad(squadBPDir, opDir string) error {
	destDir := filepath.Join(opDir, ".squad")
	if err := copyDir(squadBPDir, destDir); err != nil {
		return fmt.Errorf("copy squad snapshot: %w", err)
	}
	return nil
}

// CopySkills copies resolved skills into the runtime's dotDir.
// Skill content (excluding hooks/) goes to <dotDir>/skills/<name>/.
// Skill hooks/ are merged into <dotDir>/hooks/.
func (b *BaseProvisioner) CopySkills(skillsRoot string, resolvedSkills []string, opDir string) error {
	destSkillsDir := filepath.Join(opDir, b.dotDir, "skills")
	destHooksDir := filepath.Join(opDir, b.dotDir, "hooks")
	hooksExclude := map[string]bool{"hooks": true}

	for _, skill := range resolvedSkills {
		srcSkill := filepath.Join(skillsRoot, skill)
		if _, err := os.Stat(srcSkill); err != nil {
			continue
		}

		// Copy skill content excluding hooks/
		if err := copyDirExclude(srcSkill, filepath.Join(destSkillsDir, skill), hooksExclude); err != nil {
			return fmt.Errorf("copy skill %s: %w", skill, err)
		}

		// Copy skill hooks if they exist
		srcHooks := filepath.Join(srcSkill, "hooks")
		if info, err := os.Stat(srcHooks); err == nil && info.IsDir() {
			if err := copyDir(srcHooks, destHooksDir); err != nil {
				return fmt.Errorf("copy hooks for skill %s: %w", skill, err)
			}
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
