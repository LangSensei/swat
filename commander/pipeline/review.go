package pipeline

import "log"

// Review performs LLM-based health check.
func Review() {
	// TODO: collect operation states + recent logs
	// TODO: call LLM to analyze and suggest actions
	// TODO: execute actions (retry, cancel, reassign, notify)
	log.Println("[review] LLM review triggered — not yet implemented")
}
