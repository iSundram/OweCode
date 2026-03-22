package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/iSundram/OweCode/internal/agent"
	"github.com/iSundram/OweCode/internal/ai"
	anthropicProvider "github.com/iSundram/OweCode/internal/ai/anthropic"
	"github.com/iSundram/OweCode/internal/config"
	"github.com/iSundram/OweCode/internal/session"
	"github.com/iSundram/OweCode/internal/tools"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	cfg := config.Default()
	cfg.Provider = "anthropic"
	cfg.Model = "claude-sonnet-4-6"
	cfg.Providers["anthropic"] = config.ProviderConfig{APIKey: "test-key"}
	sess := session.New()
	reg := tools.NewRegistry()
	ag := agent.New(cfg, anthropicProvider.New(ai.ProviderConfig{
		APIKey:       cfg.Providers["anthropic"].APIKey,
		DefaultModel: cfg.Model,
	}), sess, reg)
	app := NewApp(cfg, ag, sess, "")
	return app
}

func TestSwitchProviderRejectsUnknown(t *testing.T) {
	app := newTestApp(t)
	if err := app.switchProvider("unknown-provider", ""); err == nil {
		t.Fatalf("expected error for unknown provider")
	}
}

func TestSlashProviderSwitchesProviderAndDefaultModel(t *testing.T) {
	app := newTestApp(t)
	app.handleSlashCommand("/provider deepseek")
	if app.cfg.Provider != "deepseek" {
		t.Fatalf("expected provider deepseek, got %s", app.cfg.Provider)
	}
	if app.cfg.Model != "deepseek-chat" {
		t.Fatalf("expected default deepseek model, got %s", app.cfg.Model)
	}
}

func TestSlashModelSwitchesRuntimeModel(t *testing.T) {
	app := newTestApp(t)
	app.handleSlashCommand("/model claude-opus-4-6")
	if app.cfg.Model != "claude-opus-4-6" {
		t.Fatalf("expected model switch, got %s", app.cfg.Model)
	}
}

func TestSlashAPIAndBaseURLCommandsSetProviderConfig(t *testing.T) {
	app := newTestApp(t)
	app.handleSlashCommand("/api-key abc123")
	app.handleSlashCommand("/base-url https://example.local/v1")
	pc := app.cfg.Providers[app.cfg.Provider]
	if pc.APIKey != "abc123" {
		t.Fatalf("expected api key update, got %q", pc.APIKey)
	}
	if pc.BaseURL != "https://example.local/v1" {
		t.Fatalf("expected base url update, got %q", pc.BaseURL)
	}
}

func TestSlashProviderScopedConfigCommands(t *testing.T) {
	app := newTestApp(t)
	app.handleSlashCommand("/provider-api-key xai xyz")
	app.handleSlashCommand("/provider-base-url xai https://api.x.ai/v1")
	pc := app.cfg.Providers["xai"]
	if pc.APIKey != "xyz" {
		t.Fatalf("expected provider api key update, got %q", pc.APIKey)
	}
	if pc.BaseURL != "https://api.x.ai/v1" {
		t.Fatalf("expected provider base url update, got %q", pc.BaseURL)
	}
}

func TestSlashModelAPIKeyCommands(t *testing.T) {
	app := newTestApp(t)
	app.handleSlashCommand("/model-api-key model-secret")
	pc := app.cfg.Providers[app.cfg.Provider]
	mc, ok := pc.Models[app.cfg.Model]
	if !ok {
		t.Fatalf("expected model config to exist")
	}
	if mc.APIKey != "model-secret" {
		t.Fatalf("expected model api key update, got %q", mc.APIKey)
	}

	app.handleSlashCommand("/provider-model-api-key xai grok-4.20-reasoning xai-model-secret")
	xai := app.cfg.Providers["xai"]
	xaiModel, ok := xai.Models["grok-4.20-reasoning"]
	if !ok {
		t.Fatalf("expected xai model config to exist")
	}
	if xaiModel.APIKey != "xai-model-secret" {
		t.Fatalf("expected provider-model api key update, got %q", xaiModel.APIKey)
	}
}

func TestHandleAgentEventDoneSkipsDuplicateAfterStreaming(t *testing.T) {
	app := newTestApp(t)
	app.handleAgentEvent(agent.Event{Type: agent.EventToken, Payload: "hello"})
	before := app.conversation.MessageCount()
	app.handleAgentEvent(agent.Event{Type: agent.EventDone, Payload: "hello"})
	after := app.conversation.MessageCount()
	if after != before {
		t.Fatalf("expected no duplicate message after streamed done, before=%d after=%d", before, after)
	}
}

func TestHandleAgentEventDoneAddsMessageWhenNoStreaming(t *testing.T) {
	app := newTestApp(t)
	app.handleAgentEvent(agent.Event{Type: agent.EventDone, Payload: "standalone"})
	last, ok := app.conversation.LastMessage()
	if !ok {
		t.Fatalf("expected a message")
	}
	if last.Role != "assistant" || last.Content == "" {
		t.Fatalf("expected assistant completion message, got role=%s content=%q", last.Role, last.Content)
	}
}

func TestHandleAgentEventToolDoneRendersResult(t *testing.T) {
	app := newTestApp(t)
	app.handleAgentEvent(agent.Event{Type: agent.EventToolDone, Payload: tools.Result{Content: "tool output", IsError: false}})
	last, ok := app.conversation.LastMessage()
	if !ok {
		t.Fatalf("expected a tool result message")
	}
	if last.Role != "tool_result" {
		t.Fatalf("expected tool_result role, got %s", last.Role)
	}
}

func TestHandleAgentEventToolDoneRendersError(t *testing.T) {
	app := newTestApp(t)
	app.handleAgentEvent(agent.Event{Type: agent.EventToolDone, Payload: tools.Result{Content: "boom", IsError: true}})
	last, ok := app.conversation.LastMessage()
	if !ok {
		t.Fatalf("expected an error message")
	}
	if last.Role != "assistant" || !last.IsError {
		t.Fatalf("expected assistant error message, got role=%s isError=%v", last.Role, last.IsError)
	}
}

func TestToolResultTruncatesWhenNotInReviewMode(t *testing.T) {
	app := newTestApp(t)
	long := ""
	for i := 0; i < 700; i++ {
		long += "a"
	}
	app.handleAgentEvent(agent.Event{Type: agent.EventToolDone, Payload: tools.Result{Content: long, IsError: false}})
	last, ok := app.conversation.LastMessage()
	if !ok || last.Role != "tool_result" {
		t.Fatalf("expected tool_result")
	}
	if len(last.Content) >= len(long) {
		t.Fatalf("expected truncated content")
	}
}

func TestToolResultNotTruncatedInReviewMode(t *testing.T) {
	app := newTestApp(t)
	app.conversation.SetReviewMode(true)
	long := ""
	for i := 0; i < 700; i++ {
		long += "b"
	}
	app.handleAgentEvent(agent.Event{Type: agent.EventToolDone, Payload: tools.Result{Content: long, IsError: false}})
	last, ok := app.conversation.LastMessage()
	if !ok || last.Role != "tool_result" {
		t.Fatalf("expected tool_result")
	}
	if last.Content != long {
		t.Fatalf("expected full content in review mode")
	}
}

func TestCtrlRTogglesReviewMode(t *testing.T) {
	app := newTestApp(t)
	if app.conversation.ReviewMode() {
		t.Fatalf("expected review mode off initially")
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR})
	if !app.conversation.ReviewMode() {
		t.Fatalf("expected review mode on after ctrl+r")
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR})
	if app.conversation.ReviewMode() {
		t.Fatalf("expected review mode off after second ctrl+r")
	}
}

func TestPersistProjectConfigWritesRepoOwecodeConfig(t *testing.T) {
	app := newTestApp(t)
	repo := t.TempDir()

	origWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWD) }()
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	app.cfg.Provider = "openai"
	app.cfg.Model = "gpt-5.4"
	app.ensureProviderConfig("openai")
	pc := app.cfg.Providers["openai"]
	pc.APIKey = "sk-test"
	app.cfg.Providers["openai"] = pc

	if err := app.persistProjectConfig(); err != nil {
		t.Fatalf("persistProjectConfig: %v", err)
	}

	path := filepath.Join(repo, ".owecode", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("persisted config is empty")
	}
}
