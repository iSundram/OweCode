# OweCode

> A best-in-class AI coding agent for the terminal, written in Go.

OweCode is a powerful, open-source AI coding agent designed to live where developers spend their most productive hours: the terminal. Built for speed, portability, and deep system integration, OweCode transforms your CLI into a high-agency development environment that understands your code as deeply as you do.

---

## Core Philosophy

OweCode is built on three foundational pillars that define its behavior and evolution:

- **Terminal-Native Efficiency**: Developed in Go as a single static binary, OweCode offers zero-dependency installation and instant startup. It treats the terminal as a first-class citizen, leveraging high-performance TUI components for rich interaction without leaving your shell.
- **Safety-First Agency**: We believe agents should be powerful but predictable. OweCode implements OS-native sandboxing, path-blocking for sensitive directories (like `.git`, `.ssh`), and mandatory Git-backed versioning for all code modifications. It doesn't just edit; it preserves your work's integrity.
- **Deep Code Intelligence**: By integrating directly with Language Server Protocol (LSP) and Model Context Protocol (MCP), OweCode moves beyond simple text completion. It understands symbol relationships, diagnostics, and project-wide context, acting as a true "code-aware" partner.

---

## Quickstart

```bash
# Install (requires Go 1.22+)
go install github.com/iSundram/OweCode/cmd/owecode@latest

# Start an interactive session
owecode

# Ask a specific question or request a change
owecode "explain the internal/agent package"
owecode --mode edit "add unit tests for internal/tools/filesystem"
```

---

## Suggested Workflows

Rather than listing features, we suggest several ways OweCode can become an indispensable part of your development loop:

- **Context-Aware Refactoring**: Use OweCode to perform complex, multi-file refactors. Because it understands your project structure and leverages LSP, it can ensure that changes remain idiomatically consistent and syntactically correct.
- **Automated Defensive Coding**: Suggest OweCode to analyze your existing functions and generate robust unit tests, especially for edge cases that are often overlooked.
- **System-Level Debugging**: Leverage its sandboxed shell access to have the agent run tests, analyze logs, and identify the root cause of failures within a controlled environment.
- **Rapid Prototyping**: Start with a high-level `plan` mode to scaffold new modules or features, then switch to `edit` mode to refine the implementation turn-by-turn.

---

## Vision & Future Directions

We suggest OweCode is just the beginning of a new era of terminal-native agents. Our roadmap includes:

- **Deeper LSP Synergy**: Moving from simple diagnostics to proactive refactoring suggestions based on real-time code analysis.
- **Expanded MCP Ecosystem**: Building a library of specialized MCP servers for deeper integration with cloud providers, database engines, and CI/CD pipelines.
- **Collaborative Agentic Loops**: Enabling multiple agents to collaborate on complex tasks, each specialized in different domains (e.g., security analysis vs. feature implementation).
- **Offline-First Intelligence**: Enhancing support for local-first models (via Ollama and others) to ensure your code remains private and your agent remains fast even without an internet connection.

---

## Project Structure

```
owecode/
├── cmd/
│   ├── owecode/          # CLI entry point
│   └── installer/        # TUI installer
├── internal/
│   ├── agent/            # Agent loop, tool dispatch (Edit/Plan modes)
│   ├── ai/               # Multi-provider AI orchestration
│   ├── installer/        # Installer logic and TUI
│   ├── lsp/              # LSP client integration
│   ├── mcp/              # Model Context Protocol support
│   ├── sandbox/          # OS-native security layers
│   ├── session/          # Session persistence and checkpoints
│   ├── tools/            # Atomic filesystem, shell, and git tools
│   └── tui/              # Bubble Tea + Lip Gloss interface
├── docs/                 # Detailed architectural and design documentation
├── go.mod                # Go module definition
└── README.md             # This document
```

---

## License

Apache 2.0 — see [LICENSE](LICENSE)
