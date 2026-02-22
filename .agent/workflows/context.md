---
description: Project context and documentation for AI assistant
---

# Project Context

This is a Golang stock trading project that integrates with DNSE API.

## Important Documentation

Always read these files before making changes:

1. **docs/PROJECT.md** - Project structure and features
2. **docs/IMPLEMENTATION_PLAN.md** - API endpoints and conventions
3. **docs/TASKS.md** - Current task status

## Key Architecture Decisions

1. **API Separation**: `info_api.go` (no auth) vs `trading_api.go` (needs tradingToken)
2. **Auth Flow**: Login → OTP → Trading Token
3. **JSON Convention**: lowerCamelCase for all API payloads
4. **OTP**: Valid 2 minutes, single use, auto-fetched via IMAP

## Common Commands

```bash
# Build
go build ./...

# Run
go run cmd/app/main.go

# Test
go test ./...
```
