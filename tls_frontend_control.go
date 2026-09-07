package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"waf-proxy/internal/tlsfront"
)

const defaultTLSFrontendStatusPath = "/run/waf-tls-frontend/status.json"

type tlsFrontendPublisher struct {
	path       string
	statusPath string
	log        *slog.Logger
}

func newTLSFrontendPublisher(log *slog.Logger) *tlsFrontendPublisher {
	path := strings.TrimSpace(os.Getenv("WAF_TLS_FRONTEND_CONTROL"))
	statusPath := strings.TrimSpace(os.Getenv("WAF_TLS_FRONTEND_STATUS"))
	if statusPath == "" {
		statusPath = defaultTLSFrontendStatusPath
	}
	return &tlsFrontendPublisher{path: path, statusPath: statusPath, log: log}
}

func (p *tlsFrontendPublisher) enabled() bool { return p != nil && p.path != "" }

func tlsFrontendLiveConfig(cfg Config) tlsfront.LiveConfig {
	return tlsfront.LiveConfig{
		TLSAcceleration:   cfg.TLSAcceleration,
		Sites:             tlsFrontendSites(cfg),
		ReadTimeoutSec:    cfg.ReadTimeoutSec,
		IdleTimeoutSec:    cfg.IdleTimeoutSec,
		BackendTimeoutSec: cfg.BackendTimeoutSec,
		PublishedAt:       time.Now().UTC(),
	}
}

func (p *tlsFrontendPublisher) preflight(cfg Config) error {
	if !tlsfront.FrontendEnabled(cfg.TLSAcceleration) {
		return nil
	}
	if !p.enabled() {
		return errors.New("tls_acceleration.mode=frontend requires WAF_TLS_FRONTEND_CONTROL (install/enable waf-tls-frontend.service)")
	}
	live := tlsFrontendLiveConfig(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	runtimeDir := filepath.Dir(p.statusPath)
	if runtimeDir == "." || runtimeDir == "/" {
		runtimeDir = "/run/waf-tls-frontend"
	}
	_, _, err := tlsfront.Preflight(ctx, live, runtimeDir)
	return err
}

func (p *tlsFrontendPublisher) publish(cfg Config) error {
	if !p.enabled() {
		return nil
	}
	if !filepath.IsAbs(p.path) {
		return errors.New("WAF_TLS_FRONTEND_CONTROL must be an absolute path")
	}
	live := tlsFrontendLiveConfig(cfg)
	if err := live.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(live, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	dir := filepath.Dir(p.path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create TLS frontend control directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".tls-frontend-live-*")
	if err != nil {
		return fmt.Errorf("create TLS frontend control temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o640); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, p.path); err != nil {
		return fmt.Errorf("publish TLS frontend live config: %w", err)
	}
	return nil
}

type tlsFrontendRuntimeStatus struct {
	Mode      string            `json:"mode,omitempty"`
	Active    bool              `json:"active"`
	NginxPID  int               `json:"nginx_pid,omitempty"`
	Resolved  tlsfront.Resolved `json:"resolved"`
	Probe     tlsfront.Probe    `json:"probe"`
	LastError string            `json:"last_error,omitempty"`
	UpdatedAt time.Time         `json:"updated_at,omitempty"`
	Version   string            `json:"version,omitempty"`
	Commit    string            `json:"commit,omitempty"`
}

func (p *tlsFrontendPublisher) status() tlsFrontendRuntimeStatus {
	if p == nil || p.statusPath == "" {
		return tlsFrontendRuntimeStatus{}
	}
	b, err := os.ReadFile(p.statusPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) && p.log != nil {
			p.log.Debug("TLS frontend status read failed", "err", err)
		}
		return tlsFrontendRuntimeStatus{}
	}
	var st tlsFrontendRuntimeStatus
	if err := json.Unmarshal(b, &st); err != nil {
		if p.log != nil {
			p.log.Debug("TLS frontend status decode failed", "err", err)
		}
		return tlsFrontendRuntimeStatus{LastError: "invalid frontend status file"}
	}
	return st
}
