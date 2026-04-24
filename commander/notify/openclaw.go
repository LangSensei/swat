package notify

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// loadDotEnv reads ~/.swat/.env and sets values via os.Setenv as fallback.
// Real environment variables take precedence — only keys not already set are
// applied. The file format is KEY=VALUE per line; blank lines and lines
// starting with # are skipped.
func loadDotEnv() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	f, err := os.Open(filepath.Join(home, ".swat", ".env"))
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		// Real env vars take precedence — only set if not already present.
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// readOpenClawConfig reads ~/.openclaw/openclaw.json and extracts gateway
// configuration (port and auth token). Returns a zero-value config (no error)
// if the file does not exist or cannot be parsed.
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
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return cfg
	}

	cfg.Token = raw.Gateway.Auth.Token
	if p := raw.Gateway.Port.String(); p != "" {
		cfg.Port = p
	}

	return cfg
}

// newOpenClawNotifier loads ~/.swat/.env as fallback, reads gateway config from
// ~/.openclaw/openclaw.json, then lets env vars override any values.
// Errors if any of the 4 required values (port, token, target, channel) is empty.
func newOpenClawNotifier() (*OpenClawNotifier, error) {
	loadDotEnv()

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
		return nil, fmt.Errorf("openclaw: gateway token is required (set OPENCLAW_GATEWAY_TOKEN in ~/.swat/.env or as env var, or configure gateway.auth.token in ~/.openclaw/openclaw.json)")
	}
	if cfg.Port == "" {
		return nil, fmt.Errorf("openclaw: gateway port is required (set OPENCLAW_GATEWAY_PORT in ~/.swat/.env or as env var, or configure gateway.port in ~/.openclaw/openclaw.json)")
	}
	if cfg.Target == "" {
		return nil, fmt.Errorf("openclaw: notify target is required (set OPENCLAW_NOTIFY_TARGET in ~/.swat/.env — e.g. your Telegram chat ID or Discord DM channel ID)")
	}
	if cfg.Channel == "" {
		return nil, fmt.Errorf("openclaw: notify channel is required (set OPENCLAW_NOTIFY_CHANNEL in ~/.swat/.env — e.g. telegram, discord, signal)")
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
