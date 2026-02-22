# Capability: Purchase Recording

## ADDED Requirements

### Requirement: Users can record stock purchases via Telegram command

**Priority:** High  
**Rationale:** Provides simple, accessible way for users to log purchases without leaving Telegram interface

#### Scenario: Record purchase with automatic date

**Given** a user is authenticated with the Telegram bot  
**When** the user sends `/buy VNM 100 85000`  
**Then** the system creates a position record with:

- `symbol` = "VNM"
- `quantity` = 100
- `entry_price` = 85000
- `entry_date` = current date
- `user_id` = authenticated user's ID
- `is_closed` = false

**And** the bot sends a confirmation message: "✅ Recorded purchase: 100 shares of VNM @ 85,000 VND on [current date]"

#### Scenario: Record purchase with specific date

**Given** a user wants to log a historical purchase  
**When** the user sends `/buy HPG 500 24500 2026-01-25`  
**Then** the system creates a position record with:

- `symbol` = "HPG"
- `quantity` = 500
- `entry_price` = 24500
- `entry_date` = 2026-01-25
- `user_id` = authenticated user's ID
- `is_closed` = false

**And** the bot confirms: "✅ Recorded purchase: 500 shares of HPG @ 24,500 VND on 2026-01-25"

#### Scenario: Reject invalid purchase command

**Given** a user sends an invalid command  
**When** the user sends `/buy VNM` (missing quantity and price)  
**Then** the bot responds with usage help:

```
Usage: /buy <symbol> <quantity> <price> [date]

Examples:
  /buy VNM 100 85000
  /buy VNM 100 85000 2026-01-25

Parameters:
  symbol - Stock symbol (e.g., VNM, HPG)
  quantity - Number of shares (positive integer)
  price - Purchase price per share (positive number)
  date - Optional purchase date (YYYY-MM-DD format, defaults to today)
```

#### Scenario: Validate numeric inputs

**Given** the user provides invalid numeric values  
**When** the user sends `/buy VNM -100 85000`  
**Then** the bot responds: "❌ Invalid quantity. Must be a positive number."

**When** the user sends `/buy VNM 100 -5000`  
**Then** the bot responds: "❌ Invalid price. Must be a positive number."

**When** the user sends `/buy VNM abc 85000`  
**Then** the bot responds: "❌ Invalid quantity. Must be a number."

#### Scenario: Validate date format

**Given** the user provides a date  
**When** the user sends `/buy VNM 100 85000 25-01-2026` (wrong format)  
**Then** the bot responds: "❌ Invalid date format. Please use YYYY-MM-DD (e.g., 2026-01-25)"

**When** the user sends `/buy VNM 100 85000 2026-13-45` (invalid date)  
**Then** the bot responds: "❌ Invalid date. Please check the date and try again."

### Requirement: Multiple purchases of same stock are tracked separately

**Priority:** High  
**Rationale:** Users need complete transaction history to track dollar-cost averaging and understand their position building

#### Scenario: Record multiple purchases of same stock

**Given** a user has purchased VNM twice  
**When** the user sends:

1. `/buy VNM 100 80000 2026-01-20`
2. `/buy VNM 200 85000 2026-02-01`

**Then** the database contains two separate position records:

- Position 1: 100 shares @ 80,000 on 2026-01-20
- Position 2: 200 shares @ 85,000 on 2026-02-01

**And** both positions have `is_closed` = false
**And** both have the same `user_id` and `symbol`

### Requirement: Help documentation includes /buy command

**Priority:** Medium  
**Rationale:** Users need to discover the command and understand its usage

#### Scenario: User views help text

**Given** a user wants to learn available commands  
**When** the user sends `/help`  
**Then** the response includes a Portfolio section with:

```
Portfolio:
/buy <symbol> <qty> <price> [date] - Record purchase
/positions - List active positions
/position <symbol> - Position details
```

**And** the help text shows an example: `/buy VNM 100 85000`
