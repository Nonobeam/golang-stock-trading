# Change: Migrate Telegram to Multi-User Support

## Why

Currently, the system uses a single global `TELEGRAM_CHAT_ID` environment variable that restricts the application to serving only one user. This creates a bottleneck for scaling and prevents multiple users from receiving personalized notifications and interacting with their own trading accounts via Telegram. The `user_config` table already has a `telegram_chat_id` field, but it's not being used—instead, all notifications go to the hardcoded chat ID from configuration.

## What Changes

- **BREAKING**: Remove `TELEGRAM_CHAT_ID` environment variable from configuration
- Modify Telegram bot service to identify users by `chat_id` from incoming messages
- Add user lookup/creation logic based on `telegram_chat_id`
- Update notification flows to query the database for user-specific chat IDs
- Modify ML service alerter to support multi-user notifications (query DB or accept parameters)
- Remove hardcoded chat ID from bot initialization and message authorization
- Update all places that reference the global chat ID to use database lookups

## Impact

### Affected Components

**Go Service:**

- `internal/config/config.go` - Remove `TelegramChatID` field
- `internal/service/telegram/bot_service.go` - Remove global `chatID`, add user context
- `cmd/app/main.go` - Remove chat ID initialization check
- `.env` - Remove `TELEGRAM_CHAT_ID` variable

**ML Service:**

- `ml-service/monitoring/alerter.py` - Add multi-user support (DB query + parameter)

**Database:**

- `internal/db/repository/user_config_repository.go` - Add methods to find/create users by chat ID

### Breaking Changes

- **BREAKING**: `TELEGRAM_CHAT_ID` environment variable will no longer be used
- **BREAKING**: Bot initialization no longer requires pre-configured chat ID
- **BREAKING**: Existing deployments must migrate to database-driven user management

### Benefits

- ✅ Supports multiple concurrent users
- ✅ Each user receives personalized notifications
- ✅ Users are automatically identified from their Telegram chat
- ✅ Leverages existing `telegram_chat_id` field in database
- ✅ More scalable architecture
