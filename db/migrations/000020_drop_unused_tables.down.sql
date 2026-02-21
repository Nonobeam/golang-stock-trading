-- Recreate vnindex_daily table
CREATE TABLE IF NOT EXISTS "stock-trading".vnindex_daily (
    date DATE NOT NULL PRIMARY KEY,
    value DECIMAL(10,2),
    change DECIMAL(10,2),
    volume BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Recreate trade_journal table
CREATE TABLE IF NOT EXISTS "stock-trading".trade_journal (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    position_id UUID REFERENCES "stock-trading".positions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    
    -- Pre-trade
    pre_trade_plan TEXT,
    pre_trade_checklist JSONB,
    
    -- During trade
    trade_notes TEXT,
    emotional_state VARCHAR(50),
    
    -- Post-trade
    post_trade_review TEXT,
    lessons_learned TEXT,
    mistakes TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_trade_journal_position ON "stock-trading".trade_journal(position_id);
CREATE INDEX idx_trade_journal_user ON "stock-trading".trade_journal(user_id);

-- Re-create the trigger for trade_journal
CREATE TRIGGER update_trade_journal_updated_at
    BEFORE UPDATE ON "stock-trading".trade_journal
    FOR EACH ROW
    EXECUTE FUNCTION "stock-trading".update_updated_at_column();
