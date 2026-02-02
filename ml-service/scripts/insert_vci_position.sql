-- Insert VCI Position
-- User's current VCI position documented for ML system integration
-- Date: 2026-02-02
-- Entry: 2026-01-27

INSERT INTO positions (
    user_id,
    symbol,
    entry_date,
    entry_price,
    quantity,
    stop_loss,
    target_1,
    target_2,
    target_3,
    signal_type,
    notes,
    is_closed
) VALUES (
    1,                                      -- user_id (single user system)
    'VCI',                                  -- symbol
    '2026-01-27',                          -- entry_date
    36850.00,                              -- entry_price (VND)
    100,                                    -- quantity (shares)
    35100.00,                              -- stop_loss (4.75% below entry)
    39500.00,                              -- target_1 (7.2% profit)
    42000.00,                              -- target_2 (14% profit)
    45000.00,                              -- target_3 (22% profit)
    'MANUAL',                              -- signal_type (manually entered position)
    'Initial position documented for ML system integration', -- notes
    FALSE                                   -- is_closed (position still active)
)
ON CONFLICT DO NOTHING;  -- Prevent duplicate insertion if run multiple times

-- Verify insertion
SELECT 
    id,
    symbol,
    entry_date,
    entry_price,
    quantity,
    stop_loss,
    target_1,
    target_2,
    target_3,
    is_closed,
    created_at
FROM positions
WHERE symbol = 'VCI' AND is_closed = FALSE;
