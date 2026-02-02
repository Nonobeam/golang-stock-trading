---
description: How to start working on this project
---

# Start Working on Golang Stock Trading Project

Before making any changes, read the project documentation:

// turbo-all

1. Read project overview
```bash
cat docs/PROJECT.md
```

2. Read implementation plan
```bash
cat docs/IMPLEMENTATION_PLAN.md
```

3. Read current tasks
```bash
cat docs/TASKS.md
```

4. Check current build status
```bash
go build ./...
```

5. Run the application (for testing)
```bash
go run cmd/app/main.go
```

## Key Files to Know

- `internal/config/config.go` - Configuration
- `internal/service/auth/auth_service.go` - Auth flow
- `internal/api/info_api.go` - Info APIs
- `internal/api/trading_api.go` - Trading APIs
- `internal/websocket/` - Real-time data
