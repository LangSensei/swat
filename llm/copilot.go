package llm

import "fmt"

// CopilotClient uses GitHub Copilot token for LLM calls
type CopilotClient struct {
	TokenPath string
}

// NewCopilotClient creates a new CopilotClient
func NewCopilotClient(tokenPath string) *CopilotClient {
	return &CopilotClient{TokenPath: tokenPath}
}

// Complete sends a prompt to the LLM and returns the response
func (c *CopilotClient) Complete(prompt string) (string, error) {
	// TODO: read token from TokenPath
	// TODO: exchange for short-lived access token
	// TODO: call Copilot API endpoint
	return "", fmt.Errorf("copilot LLM not yet implemented")
}
