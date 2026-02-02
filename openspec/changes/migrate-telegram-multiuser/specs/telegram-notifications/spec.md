## ADDED Requirements

### Requirement: User Identification via Chat ID

The Telegram bot service SHALL identify users by their Telegram `chat_id` from incoming messages and automatically create user records if they don't exist.

#### Scenario: New user sends first message

- **WHEN** a user sends their first message to the bot
- **THEN** the system SHALL extract the `chat_id` from the message
- **AND** create a new `user_config` record with that `telegram_chat_id`
- **AND** assign default trading configuration values
- **AND** send a welcome message to the user

#### Scenario: Existing user sends message

- **WHEN** an existing user sends a message to the bot
- **THEN** the system SHALL look up the user by `telegram_chat_id`
- **AND** use that user's context for all operations
- **AND** NOT create duplicate user records

### Requirement: Per-User OTP Management

The system SHALL manage OTP requests independently for each user to support concurrent authentication flows.

#### Scenario: Single user requests OTP

- **WHEN** a user triggers an OTP request
- **THEN** the system SHALL create a user-specific OTP channel
- **AND** wait for OTP input from that specific chat ID
- **AND** associate the OTP with the correct user session

#### Scenario: Multiple users request OTP concurrently

- **WHEN** multiple users request OTP at the same time
- **THEN** each user SHALL have an independent OTP channel
- **AND** OTP from user A SHALL NOT be delivered to user B
- **AND** each user SHALL have independent timeout handling

### Requirement: User-Specific Notifications

The notification system SHALL route messages to specific users based on database lookups of their `telegram_chat_id`.

#### Scenario: Send notification to specific user

- **WHEN** the system needs to send a notification to a user
- **THEN** it SHALL query the `user_config` table for that user's `telegram_chat_id`
- **AND** send the message to that specific chat ID
- **AND** NOT broadcast to other users

#### Scenario: User does not exist

- **WHEN** attempting to notify a user that doesn't exist in the database
- **THEN** the system SHALL log an error
- **AND** NOT send the notification
- **AND** NOT create a user record automatically

### Requirement: Broadcast Notifications

The system SHALL support sending system-wide notifications to all active users.

#### Scenario: Broadcast to all users

- **WHEN** the system needs to send a broadcast notification
- **THEN** it SHALL query all active users from `user_config`
- **AND** send the message to each user's `telegram_chat_id`
- **AND** log any delivery failures per user

### Requirement: Database-Driven User Lookup

The `UserConfigRepository` SHALL provide methods to find and create users by Telegram chat ID.

#### Scenario: Find user by chat ID

- **WHEN** querying for a user by chat ID
- **THEN** return the matching `UserConfig` if found
- **AND** return an error if not found
- **AND** NOT create a new user

#### Scenario: Create user for chat ID

- **WHEN** creating a new user for a chat ID
- **THEN** insert a new record with the provided `telegram_chat_id`
- **AND** set default values for other configuration fields
- **AND** return the created `UserConfig`
- **AND** fail if the chat ID already exists (constraint violation)

#### Scenario: Get or create user by chat ID

- **WHEN** using the convenience method
- **THEN** return existing user if found
- **AND** create and return new user if not found
- **AND** ensure atomicity to prevent race conditions

## REMOVED Requirements

### Requirement: Global Chat ID Configuration

**Reason**: Migrating to multi-user support where chat IDs are stored in database per user.

**Migration**: Remove `TELEGRAM_CHAT_ID` from environment variables and configuration struct. Users are now identified by their Telegram chat ID when they message the bot.

The system previously required a global `TELEGRAM_CHAT_ID` environment variable that restricted the bot to a single user. This requirement is removed in favor of database-driven user identification.

#### Scenario: Bot initialization with chat ID (REMOVED)

This scenario is no longer valid. Bot now initializes without a pre-configured chat ID.

#### Scenario: Message authorization by chat ID (REMOVED)

The bot previously rejected messages from chat IDs that didn't match the configured value. This authorization check is removed to support multi-user access.

## MODIFIED Requirements

### Requirement: ML Service Alert Delivery

The ML monitoring service SHALL support sending alerts to multiple users by querying the database or accepting chat IDs as parameters.

#### Scenario: ML service sends alert independently

- **WHEN** the ML service detects an issue requiring notification
- **THEN** it SHALL query the database for all active user chat IDs
- **AND** send the alert to each user's `telegram_chat_id`
- **AND** log delivery status per user

#### Scenario: ML service receives chat ID from Go service

- **WHEN** the Go service calls the ML alerter with specific chat IDs
- **THEN** the ML service SHALL send alerts to those provided chat IDs
- **AND** NOT query the database
- **AND** support single or multiple chat IDs in the same call

#### Scenario: ML service alert to specific user

- **WHEN** sending an alert for a specific user context
- **THEN** the ML service SHALL accept a `chat_id` parameter
- **AND** send the alert only to that chat ID
- **AND** NOT broadcast to all users
