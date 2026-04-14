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
