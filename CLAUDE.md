# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CostHandler is a personal expense tracker: a Telegram bot that receives free-text messages, classifies them via OpenAI into expense categories, stores them in SQLite, and exposes a JSON API. This is a learning project for Go.

## Architecture

Three Go modules managed via a `go.work` workspace:

- **CostHandler_mcp** — Data layer. SQLite database, repository pattern, expense service (validation/business logic), HTTP handlers for the JSON API. This is the only module with tests currently.
- **CostHandler_agent** — AI classification layer. Wraps the OpenAI API to parse free-text messages into structured expenses (amount, category, description, confidence).
- **CostHandler_bot** — Telegram bot. Receives user messages, delegates to the agent for classification, wires everything together in `cmd/main.go`. This is the application entry point.

Dependency flow: `bot → agent → mcp` (bot imports both agent and mcp; agent is standalone; mcp is standalone).

## Build and Run

```bash
# Run the full application (bot + HTTP server)
cd CostHandler_bot && go run ./cmd/main.go

# Build for deployment
cd CostHandler_bot && CGO_ENABLED=0 go build -o costhandler ./cmd/main.go
```

## Tests

```bash
# Run all tests in a specific module
cd CostHandler_mcp && go test ./... -v

# Run a single test
cd CostHandler_mcp && go test ./service -run TestCreate -v

# Run tests via Makefile
cd CostHandler_mcp && make test
cd CostHandler_agent && make test
```

Tests use SQLite `:memory:` databases and table-driven test patterns.

## Environment Variables

Required in `.env` (sourced before running):
- `TELEGRAM_TOKEN` — from @BotFather
- `OPENAI_API_KEY` — OpenAI API key
- `DB_PATH` — SQLite file path (defaults handled in config)
- `SERVER_PORT` — HTTP API port

## Valid Expense Categories

restaurantes, supermercado, transporte, entretenimiento, salud, hogar, servicios, educacion, ropa, otros

## Language

Code comments and documentation are in Spanish. Follow this convention.
