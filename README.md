# OweCode

> A best-in-class AI coding agent for the terminal, written in Go.

OweCode is a powerful, open-source AI coding agent that runs entirely in your terminal. Built in Go for maximum performance and portability, it brings the best features from leading AI coding tools into a single, cohesive experience — with a rich TUI, multi-provider AI support, and deep code intelligence.

---

## Why OweCode?

| Feature | OweCode | OpenAI Codex CLI | Gemini CLI | OpenCode |
|---|---|---|---|---|
| Language | **Go** | Rust / TypeScript | TypeScript | Go/TypeScript |
| TUI | **Rich Bubble Tea TUI** | Basic | Basic | Good |
| Multi-provider | ✅ | Limited | Google only | ✅ |
| LSP Support | ✅ | ❌ | ❌ | ✅ |
| Sandboxing | ✅ | ✅ | ✅ | ❌ |
| MCP Support | ✅ | ❌ | ✅ | ✅ |
| Offline/Local Models | ✅ | Via Ollama | ❌ | ✅ |
| Single Binary | ✅ | ✅ | ❌ | ❌ |
| Context Files | ✅ (OWECODE.md) | ✅ (AGENTS.md) | ✅ (GEMINI.md) | ✅ |
| Session Checkpointing | ✅ | ❌ | ✅ | ✅ |

---

## Quickstart

```bash
# Install
go install github.com/iSundram/OweCode/cmd/owecode@latest

# Or via install script
curl -fsSL https://owecode.dev/install | bash

# Run interactively
owecode

# Run with a prompt
owecode "explain this codebase to me"

# Full auto mode
owecode --mode full-auto "add unit tests for all Go files"
```

---

## Key Features

- **🤖 Multi-Provider AI**: OpenAI GPT-4o, Anthropic Claude, Google Gemini, Mistral, Ollama (local), DeepSeek, and more
- **🎨 Rich Terminal UI**: Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) — syntax highlighting, diff views, progress indicators
- **🔒 Sandboxed Execution**: OS-native sandboxing (macOS Seatbelt, Linux namespaces/Docker), network-off by default in full-auto
- **📋 Three Approval Modes**: `suggest` (default), `auto-edit`, `full-auto`
- **🔍 LSP Integration**: Real-time diagnostics, go-to-definition, hover docs while the agent works
- **🔌 MCP Support**: Extend with any Model Context Protocol server
- **💾 Session Checkpointing**: Save, resume, and branch conversations
- **📁 Context Files (OWECODE.md)**: Per-project persistent instructions and memory
- **⚡ Single Static Binary**: No Node.js, no runtime — just one Go binary
- **🖥️ Headless/CI Mode**: Pipe-friendly, JSON output, scriptable

---

## Documentation

| Document | Description |
|---|---|
| [Research](docs/RESEARCH.md) | Deep-dive feature research from OpenAI Codex, Gemini CLI, and OpenCode |
| [Features](docs/FEATURES.md) | Best-in-class features selected for OweCode |
| [Architecture](docs/ARCHITECTURE.md) | Go project structure and component design |
| [TUI Design](docs/TUI.md) | Terminal UI layout, widgets, and interaction flows |
| [Commands](docs/COMMANDS.md) | Full CLI commands and slash-command reference |
| [Prompts](docs/PROMPTS.md) | Prompt engineering, context management, and system prompts |
| [Configuration](docs/CONFIGURATION.md) | Configuration file format, env vars, and options |
| [Security](docs/SECURITY.md) | Security model, sandboxing, and permission system |
| [Bugs](docs/BUGS.md) | Common bugs to avoid and known solutions |
| [Implementation](docs/IMPLEMENTATION.md) | Step-by-step Go implementation guide |

---

## Project Structure

```
owecode/
├── cmd/
│   └── owecode/          # CLI entry point
├── internal/
│   ├── agent/            # Agent loop, tool dispatch
│   ├── ai/               # AI provider clients (OpenAI, Gemini, Claude, Ollama...)
│   ├── lsp/              # LSP client integration
│   ├── mcp/              # Model Context Protocol client
│   ├── sandbox/          # OS sandboxing (macOS/Linux)
│   ├── session/          # Session management, checkpointing
│   ├── tools/            # Built-in tools (file, shell, search...)
│   └── tui/              # Bubble Tea TUI components
├── docs/                 # All documentation
├── go.mod
└── README.md
```

---

## License

Apache 2.0 — see [LICENSE](LICENSE)
