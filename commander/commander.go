package commander

import (
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/notify"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/pipeline"
)

// Commander is the core orchestrator
type Commander struct {
	RuntimeName   string
	NotifyBackend string
	Notifier      notify.Notifier

	iteration      int
	recentFailures int
	RetryCount     map[string]int
}

// New creates a new Commander instance
func New(runtimeName, notifyBackend string) *Commander {
	layout.EnsureDirs()

	n, err := notify.New(notifyBackend)
	if err != nil {
		log.Printf("[commander] notify init error (falling back to desktop): %v", err)
		n = &notify.DesktopNotifier{}
	}

	return &Commander{
		RuntimeName:   runtimeName,
		NotifyBackend: notifyBackend,
		Notifier:      n,
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

// ListOperations returns all operations across all squads.
func (c *Commander) ListOperations() ([]*operation.Operation, error) {
	return operation.List()
}

// BackgroundLoop runs the Commander's periodic scan.
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

func (c *Commander) scan() {
	ops, err := c.ListOperations()
	if err != nil {
		log.Printf("[scan] error: %v", err)
		return
	}
	c.recentFailures = 0
	for _, op := range ops {
		if op.Status == "active" {
			pipeline.HandleActive(op, c.Notifier)
			if op.DispatchedAt != nil && time.Since(*op.DispatchedAt) > 30*time.Minute {
				c.recentFailures++
			}
		}
	}
}

func (c *Commander) shouldReview() bool {
	if c.iteration%10 == 0 {
		return true
	}
	if c.recentFailures > 0 {
		return true
	}
	ops, _ := c.ListOperations()
	for _, op := range ops {
		if op.Status == "active" && op.DispatchedAt != nil {
			if time.Since(*op.DispatchedAt) > 30*time.Minute {
				return true
			}
		}
	}
	return false
}
