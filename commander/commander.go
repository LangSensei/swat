package commander

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/LangSensei/swat/commander/notify"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/pipeline"
)

// Commander is the core orchestrator
type Commander struct {
	SwatRoot      string
	RuntimeName   string
	NotifyBackend string
	Notifier      notify.Notifier
	Store         *operation.Store

	// Internal state for background loop
	iteration      int
	recentFailures int
	RetryCount     map[string]int
}

// New creates a new Commander instance
func New(swatRoot, runtimeName, notifyBackend string) *Commander {
	if len(swatRoot) >= 2 && swatRoot[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			swatRoot = filepath.Join(home, swatRoot[2:])
		}
	}

	// Ensure directory structure exists
	for _, dir := range []string{
		swatRoot,
		filepath.Join(swatRoot, "blueprints"),
		filepath.Join(swatRoot, "blueprints", "squads"),
		filepath.Join(swatRoot, "blueprints", "skills"),
		filepath.Join(swatRoot, "blueprints", "mcps"),
		filepath.Join(swatRoot, "squads"),
		filepath.Join(swatRoot, "squads", "_unclassified", "operations"),
	} {
		os.MkdirAll(dir, 0755)
	}

	// Initialize notifier
	n, err := notify.New(notifyBackend)
	if err != nil {
		log.Printf("[commander] notify init error (falling back to desktop): %v", err)
		n = &notify.DesktopNotifier{}
	}

	return &Commander{
		SwatRoot:      swatRoot,
		RuntimeName:   runtimeName,
		NotifyBackend: notifyBackend,
		Notifier:      n,
		Store:         operation.NewStore(swatRoot),
		RetryCount:    make(map[string]int),
	}
}

// GenerateOpID creates a unique operation identifier.
func GenerateOpID() string {
	now := time.Now().UTC()
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", now.Format("20060102"), b)
}

// ListOperations delegates to the operation Store.
func (c *Commander) ListOperations() ([]*operation.Operation, error) {
	return c.Store.List()
}
func (c *Commander) BackgroundLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.iteration++
		c.scan()
		c.CheckDue()
		if c.shouldReview() {
			pipeline.Review()
		}
	}
}

// scan checks all operations for state transitions.
func (c *Commander) scan() {
	ops, err := c.Store.List()
	if err != nil {
		log.Printf("[scan] error: %v", err)
		return
	}
	c.recentFailures = 0
	for _, op := range ops {
		if op.Status == "active" {
			pipeline.HandleActive(op, c.Store, c.Notifier)
			// Track long-running
			if op.DispatchedAt != nil && time.Since(*op.DispatchedAt) > 30*time.Minute {
				c.recentFailures++
			}
		}
	}
}

// shouldReview determines if LLM review is needed.
func (c *Commander) shouldReview() bool {
	if c.iteration%10 == 0 {
		return true
	}
	if c.recentFailures > 0 {
		return true
	}
	ops, _ := c.Store.List()
	for _, op := range ops {
		if op.Status == "active" && op.DispatchedAt != nil {
			if time.Since(*op.DispatchedAt) > 30*time.Minute {
				return true
			}
		}
	}
	return false
}
