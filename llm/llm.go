package llm

// Client is the interface for LLM calls
type Client interface {
	Complete(prompt string) (string, error)
}
