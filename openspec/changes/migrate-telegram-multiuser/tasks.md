# Implementation Tasks

## 1. Database Repository Layer

- [ ] 1.1 Add `FindUserByChatID(chatID int64)` method to `UserConfigRepository`
- [ ] 1.2 Add `CreateUserForChatID(chatID int64, defaults UserConfig)` method
- [ ] 1.3 Add `GetOrCreateUserByChatID(chatID int64)` convenience method
- [ ] 1.4 Add test coverage for new repository methods

## 2. Configuration Cleanup

- [ ] 2.1 Remove `TelegramChatID` field from `Config` struct in `internal/config/config.go`
- [ ] 2.2 Remove `viper.GetInt64("TELEGRAM_CHAT_ID")` from config loading
- [ ] 2.3 Remove `TELEGRAM_CHAT_ID=1669453901` from `.env` file
- [ ] 2.4 Update config validation to not check for chat ID

## 3. Bot Service Refactoring

- [ ] 3.1 Remove `chatID int64` field from `BotService` struct
- [ ] 3.2 Add `userConfigRepo *repository.UserConfigRepository` to `BotService` struct
- [ ] 3.3 Update `NewBotService()` to accept `userConfigRepo` parameter and remove chat ID initialization
- [ ] 3.4 Modify `handleMessage()` to extract chat ID from incoming message
- [ ] 3.5 Remove chat ID authorization checks (lines 84, 96) - accept all messages
- [ ] 3.6 Update `SendMessage()` to accept `chatID int64` parameter (or create `SendMessageToUser(userID int64, text string)`)
- [ ] 3.7 Create `getUserContext(chatID int64)` helper to get/create user from chat ID
- [ ] 3.8 Update `RequestSmartOTP()` to track OTP per user (use map[int64]chan string)

## 4. Notification System Updates

- [ ] 4.1 Update `NotifySignalDetected()` to accept user ID or lookup chat ID from DB
- [ ] 4.2 Update `NotifyStockAlert()` to support user-specific notifications
- [ ] 4.3 Update `NotifyTradeExecuted()` to support user-specific notifications
- [ ] 4.4 Update `NotifyPriceMonitorAlert()` to support user-specific notifications
- [ ] 4.5 Update `NotifyBatchSignals()` to support user-specific notifications
- [ ] 4.6 Modify notification callers to pass user context

## 5. Main Application Updates

- [ ] 5.1 Update `main.go` to remove `cfg.TelegramChatID != 0` check
- [ ] 5.2 Pass `userConfigRepo` to `telegram.NewBotService()`
- [ ] 5.3 Update bot initialization to work without pre-configured chat ID
- [ ] 5.4 Add logic to handle first-time user registration via Telegram

## 6. ML Service Integration

- [ ] 6.1 Add database connection to `ml-service/monitoring/alerter.py`
- [ ] 6.2 Add `get_user_chat_ids()` method to query all active users
- [ ] 6.3 Update `send_alert()` to accept optional `chat_id` parameter
- [ ] 6.4 Add `send_alert_to_all_users()` method for broadcast notifications
- [ ] 6.5 Add `send_alert_to_user(user_id: int, message: str)` method
- [ ] 6.6 Remove fallback to `TELEGRAM_CHAT_ID` environment variable

## 7. Testing & Validation

- [ ] 7.1 Test multi-user message handling
- [ ] 7.2 Test user auto-creation on first message
- [ ] 7.3 Test user-specific notifications
- [ ] 7.4 Test OTP flow with multiple concurrent users
- [ ] 7.5 Verify ML service can send alerts to multiple users
- [ ] 7.6 Test bot commands work for each user independently

## 8. Documentation

- [ ] 8.1 Update README with multi-user setup instructions
- [ ] 8.2 Document user registration flow
- [ ] 8.3 Add migration guide for existing single-user deployments
- [ ] 8.4 Update environment variable documentation
