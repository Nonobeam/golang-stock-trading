# Design: Telegram Multi-User Support

## Context

The current implementation uses a global `TELEGRAM_CHAT_ID` environment variable that ties the entire application to a single Telegram user. This prevents the system from scaling to multiple concurrent users. The database already has infrastructure for multi-user support via the `user_config.telegram_chat_id` field, but it's not being utilized. This change migrates from environment-based single-user configuration to database-driven multi-user identification.

## Goals / Non-Goals

### Goals

- Enable multiple users to interact with the bot simultaneously
- Automatically identify/create users based on their Telegram chat ID
- Route notifications to the correct user based on database lookups
- Support multi-user in both Go and Python (ML) services
- Remove dependency on `TELEGRAM_CHAT_ID` environment variable

### Non-Goals

- User authentication/authorization beyond Telegram chat ID
- User management UI or admin panel
- Migration of existing user data (fresh start acceptable)
- Multi-bot support (still single bot token)

## Decisions

### User Identification Strategy

**Decision**: Use Telegram `chat_id` from incoming messages as the primary user identifier.

**Why**:

- Every Telegram message includes `message.Chat.ID`
- Telegram chat IDs are unique and persistent
- No additional authentication needed—Telegram handles it
- Aligns with existing `user_config.telegram_chat_id` schema

**Implementation**:

```go
func (s *BotService) getUserContext(chatID int64) (*repository.UserConfig, error) {
    // Try to find existing user
    user, err := s.userConfigRepo.FindUserByChatID(chatID)
    if err == nil {
        return user, nil
    }

    // Create new user with defaults if not found
    return s.userConfigRepo.CreateUserForChatID(chatID, defaultConfig)
}
```

### OTP Management Per User

**Decision**: Change OTP channel from single global channel to a map of channels per chat ID.

**Current**:

```go
type BotService struct {
    otpChan chan string  // Single global channel
    chatID  int64        // Single global chat ID
}
```

**New**:

```go
type BotService struct {
    otpChans map[int64]chan string  // Per-user channels
    mu       sync.RWMutex            // Protect the map
}
```

**Why**:

- Multiple users can request OTP simultaneously
- No cross-contamination of OTP between users
- Clean per-user state management

### Notification Routing

**Decision**: Create two notification patterns:

1. **User-specific** (most notifications):

   ```go
   func (s *BotService) SendMessageToUser(userID int64, text string) error
   ```

   - Looks up `telegram_chat_id` from `user_config` table
   - Sends message to that specific chat

2. **Broadcast** (optional, for system-wide alerts):
   ```go
   func (s *BotService) BroadcastMessage(text string) error
   ```

   - Queries all active users from DB
   - Sends to each chat ID

**Why**:

- Explicit about notification intent
- Easy to understand and maintain
- Supports both personalized and broadcast use cases

### ML Service Integration

**Decision**: Hybrid approach—support both database queries and parameter passing.

**Pattern 1: Independent Functions** (query DB):

```python
def send_alert_to_all_users(self, message: str):
    chat_ids = self.get_user_chat_ids()  # Query DB
    for chat_id in chat_ids:
        self.send_alert(message, chat_id)
```

**Pattern 2: Called from Go** (receive parameter):

```python
def send_alert(self, message: str, chat_id: int = None):
    if chat_id is None:
        # Fallback or error
        logger.warning("No chat_id provided")
        return False
    # Send to specific chat_id
```

**Why**:

- Flexibility for different use cases
- ML service can operate independently (scheduled jobs)
- Go service can control specific notifications (real-time events)
- Supports both sync and async notification patterns

### Authorization Changes

**Decision**: Remove chat ID-based message filtering. Accept messages from all users.

**Current**:

```go
if update.Message.Chat.ID != s.chatID {
    logger.Warn().Msg("Unauthorized chat")
    continue
}
```

**New**: Remove this check entirely.

**Why**:

- Multi-user system should accept all messages
- Security handled by Telegram bot token (only authorized users can message)
- Each user operates in their own context

**Trade-off**: Anyone who knows the bot username can interact. Mitigation: Add user allowlist in future if needed.

## Data Model

### Existing Schema (no changes needed)

```sql
CREATE TABLE user_config (
    user_id BIGSERIAL PRIMARY KEY,
    telegram_chat_id BIGINT UNIQUE NOT NULL,
    initial_capital DECIMAL(15, 2),
    max_positions INT DEFAULT 3,
    ...
);
```

**Key Points**:

- `telegram_chat_id` is already `UNIQUE NOT NULL`
- `user_id` is the internal identifier
- Perfect for direct lookups

### Repository Methods to Add

```go
// Find existing user by chat ID
func (r *UserConfigRepository) FindUserByChatID(chatID int64) (*UserConfig, error)

// Create new user with chat ID
func (r *UserConfigRepository) CreateUserForChatID(chatID int64, defaults UserConfig) (*UserConfig, error)

// Convenience: get or create
func (r *UserConfigRepository) GetOrCreateUserByChatID(chatID int64) (*UserConfig, error)

// Get all active users (for broadcast)
func (r *UserConfigRepository) GetAllActiveUsers() ([]UserConfig, error)
```

## Migration Plan

### Phase 1: Go Service

1. Update repository with new methods
2. Refactor `BotService` to remove global chat ID
3. Update all `SendMessage` calls to use user context
4. Remove config and env variable
5. Test multi-user scenarios

### Phase 2: ML Service

1. Add database connection to alerter
2. Implement DB query methods
3. Update `send_alert` to support multiple chat IDs
4. Remove `TELEGRAM_CHAT_ID` fallback
5. Test integration with Go service

### Phase 3: Verification

1. Test with 2+ concurrent users
2. Verify OTP flow works independently
3. Verify notifications route correctly
4. Verify ML alerts reach all users

### Rollback

If issues arise:

- Revert code changes
- Re-add `TELEGRAM_CHAT_ID` to `.env`
- Redeploy previous version
- No database changes needed (schema unchanged)

## Risks / Trade-offs

### Risk: User Auto-Creation

**Risk**: Any Telegram user who messages the bot gets a database entry.

**Mitigation**:

- Add user cleanup job for inactive users
- Future: Add allowlist/blocklist
- Future: Require admin approval for new users

**Trade-off**: Simplicity now vs. security later. Acceptable for MVP.

### Risk: OTP Confusion

**Risk**: Multiple users requesting OTP at same time.

**Mitigation**:

- Per-user OTP channels (map-based)
- Clear messaging: "Your OTP request..."
- Timeout per user (not global)

### Risk: ML Service DB Connection

**Risk**: Python service needs database credentials.

**Mitigation**:

- Use same DB connection pattern as Go
- Environment variables: `DB_HOST`, `DB_USER`, etc.
- Connection pooling with psycopg2

### Trade-off: Complexity vs. Flexibility

**Current**: Simple, single-user, environment-based.
**New**: More complex, multi-user, database-driven.

**Justification**: Necessary for scaling. Database infrastructure already exists. Complexity is manageable with clear patterns.

## Open Questions

1. **User Defaults**: What default values for `initial_capital`, `max_positions`, etc.?
   - **Proposed**: Hardcode reasonable defaults, allow user to configure via `/config` command later

2. **First Message Flow**: Should we send a welcome message to new users?
   - **Proposed**: Yes, send welcome message explaining how to set up their account

3. **Inactive Users**: When to clean up unused accounts?
   - **Proposed**: Defer to future (not in this change)

4. **Admin Commands**: Do we need admin-only commands (e.g., list all users)?
   - **Proposed**: Defer to future (not in this change)
