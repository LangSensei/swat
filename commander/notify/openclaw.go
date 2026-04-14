package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// OpenClawNotifier sends notifications via the OpenClaw Gateway HTTP API.
type OpenClawNotifier struct {
	port    string
	token   string
	target  string
	channel string
}

// openclawConfig holds values extracted from ~/.openclaw/openclaw.json.
type openclawConfig struct {
	Token   string
	Port    string
	Target  string
	Channel string
}

// readOpenClawConfig reads ~/.openclaw/openclaw.json and extracts gateway and
// channel configuration. Returns a zero-value config (no error) if the file
// does not exist or cannot be parsed.
func readOpenClawConfig() openclawConfig {
	var cfg openclawConfig

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	data, err := os.ReadFile(filepath.Join(home, ".openclaw", "openclaw.json"))
	if err != nil {
		return cfg
	}

	var raw struct {
		Gateway struct {
			Port json.Number `json:"port"`
			Auth struct {
				Token string `json:"token"`
			} `json:"auth"`
		} `json:"gateway"`
		Channels map[string]struct {
			AllowFrom []string `json:"allowFrom"`
		} `json:"channels"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return cfg
	}

	cfg.Token = raw.Gateway.Auth.Token
	if p := raw.Gateway.Port.String(); p != "" {
		cfg.Port = p
	}

	// Use the first channel that has an allowFrom entry as target and channel name.
	for name, ch := range raw.Channels {
		if len(ch.AllowFrom) > 0 {
			cfg.Target = ch.AllowFrom[0]
			cfg.Channel = name
			break
		}
	}

	return cfg
}

// newOpenClawNotifier builds config from ~/.openclaw/openclaw.json first, then
// lets env vars override any values. Errors if token, target, or port is empty.
func newOpenClawNotifier() (*OpenClawNotifier, error) {
	cfg := readOpenClawConfig()

	// Env vars override config file values.
	if v := os.Getenv("OPENCLAW_GATEWAY_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("OPENCLAW_GATEWAY_PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("OPENCLAW_NOTIFY_TARGET"); v != "" {
		cfg.Target = v
	}
	if v := os.Getenv("OPENCLAW_NOTIFY_CHANNEL"); v != "" {
		cfg.Channel = v
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("openclaw: gateway token is required (set in ~/.openclaw/openclaw.json or OPENCLAW_GATEWAY_TOKEN)")
	}
	if cfg.Target == "" {
		return nil, fmt.Errorf("openclaw: notify target is required (set in ~/.openclaw/openclaw.json or OPENCLAW_NOTIFY_TARGET)")
	}
	if cfg.Port == "" {
		return nil, fmt.Errorf("openclaw: gateway port is required (set in ~/.openclaw/openclaw.json or OPENCLAW_GATEWAY_PORT)")
	}

	return &OpenClawNotifier{
		port:    cfg.Port,
		token:   cfg.Token,
		target:  cfg.Target,
		channel: cfg.Channel,
	}, nil
}

// Notify sends a notification via the OpenClaw Gateway API.
// Retries up to 2 times with a 2-second delay between attempts.
func (o *OpenClawNotifier) Notify(message string) error {
	args := map[string]string{
		"action":  "send",
		"target":  o.target,
		"message": message,
	}
	if o.channel != "" {
		args["channel"] = o.channel
	}

	payload := map[string]interface{}{
		"tool": "message",
		"args": args,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("openclaw: marshal payload: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%s/tools/invoke", o.port)

	const maxRetries = 2
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("openclaw: retry %d/%d...", attempt, maxRetries)
			time.Sleep(2 * time.Second)
		}

		lastErr = o.doPost(url, body)
		if lastErr == nil {
			return nil
		}
	}

	return fmt.Errorf("openclaw: failed after %d retries: %w", maxRetries, lastErr)
}

func (o *OpenClawNotifier) doPost(url string, body []byte) error {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response: %w (body: %s)", err, string(respBody))
	}
	if !result.OK {
		return fmt.Errorf("gateway returned ok:false (body: %s)", string(respBody))
	}

	return nil
}
