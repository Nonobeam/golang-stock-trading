-- Migration: 000013_add_position_entries_tracking
-- Description: Add position_entries table for tracking individual purchases and extend positions table with aggregate fields
-- Created: 2026-02-03

-- Table: position_entries
-- Stores individual purchase transactions for each stock position
CREATE TABLE position_entries (
    entry_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    ticker VARCHAR(10) NOT NULL,
    
    -- Transaction details
    entry_date TIMESTAMP NOT NULL,
    entry_price DECIMAL(15, 2) NOT NULL,
    shares_purchased INTEGER NOT NULL CHECK (shares_purchased > 0),
    entry_fee_paid DECIMAL(15, 2) NOT NULL DEFAULT 0,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('BUY_NEW', 'BUY_MORE')),
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key to positions (optional, for referential integrity)
    -- Note: Position may not exist yet when first entry is created
    CONSTRAINT fk_position_entry_user FOREIGN KEY (user_id) REFERENCES user_config(user_id) ON DELETE CASCADE
);

-- Indexes for position_entries
CREATE INDEX idx_position_entries_ticker ON position_entries(user_id, ticker, entry_date DESC);
CREATE INDEX idx_position_entries_date ON position_entries(entry_date);
CREATE INDEX idx_position_entries_user ON position_entries(user_id);

-- Add aggregate tracking columns to positions table
ALTER TABLE positions ADD COLUMN IF NOT EXISTS total_entries INTEGER DEFAULT 0;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS total_fees_paid DECIMAL(15, 2) DEFAULT 0;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS first_entry_date TIMESTAMP;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS last_entry_date TIMESTAMP;

-- Function: Calculate and update weighted average cost for a position
-- This function is called by trigger when position_entries changes
CREATE OR REPLACE FUNCTION update_position_average_cost()
RETURNS TRIGGER AS $$
DECLARE
    v_total_cost DECIMAL(20, 2);
    v_total_shares INTEGER;
    v_avg_cost DECIMAL(15, 2);
    v_position_id UUID;
    v_entry_count INTEGER;
    v_total_fees DECIMAL(15, 2);
    v_first_date TIMESTAMP;
    v_last_date TIMESTAMP;
BEGIN
    -- Calculate weighted average from all entries for this ticker
    SELECT 
        SUM(shares_purchased * entry_price),
        SUM(shares_purchased),
        SUM(entry_fee_paid),
        COUNT(*),
        MIN(entry_date),
        MAX(entry_date)
    INTO 
        v_total_cost,
        v_total_shares,
        v_total_fees,
        v_entry_count,
        v_first_date,
        v_last_date
    FROM position_entries
    WHERE user_id = NEW.user_id 
      AND ticker = NEW.ticker;
    
    -- Calculate average cost
    IF v_total_shares > 0 THEN
        v_avg_cost := v_total_cost / v_total_shares;
    ELSE
        v_avg_cost := 0;
    END IF;
    
    -- Find or create position record
    SELECT id INTO v_position_id
    FROM positions
    WHERE user_id = NEW.user_id 
      AND symbol = NEW.ticker 
      AND is_closed = FALSE
    LIMIT 1;
    
    IF v_position_id IS NOT NULL THEN
        -- Update existing position with new average and aggregates
        UPDATE positions
        SET entry_price = v_avg_cost,
            quantity = v_total_shares,
            total_entries = v_entry_count,
            total_fees_paid = v_total_fees,
            first_entry_date = v_first_date,
            last_entry_date = v_last_date,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = v_position_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger: Auto-update position average cost when entries change
CREATE TRIGGER trigger_update_average_cost
    AFTER INSERT ON position_entries
    FOR EACH ROW
    EXECUTE FUNCTION update_position_average_cost();

-- Comments for documentation
COMMENT ON TABLE position_entries IS 'Immutable log of individual stock purchase transactions for weighted average cost calculation';
COMMENT ON COLUMN position_entries.entry_fee_paid IS 'Transaction fee paid on this purchase (typically 0.15% of purchase value)';
COMMENT ON COLUMN position_entries.transaction_type IS 'BUY_NEW for first purchase, BUY_MORE for adding to existing position';

COMMENT ON COLUMN positions.total_entries IS 'Count of purchase transactions for this position';
COMMENT ON COLUMN positions.total_fees_paid IS 'Accumulated entry fees from all purchases (exit fee added on close)';
COMMENT ON COLUMN positions.first_entry_date IS 'Date of first purchase (used for stop-loss calculation)';
COMMENT ON COLUMN positions.last_entry_date IS 'Date of most recent purchase';
