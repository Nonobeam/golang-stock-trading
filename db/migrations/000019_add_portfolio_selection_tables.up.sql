-- Migration: 000019_add_portfolio_selection_tables
-- Description: Stock universe registry and weekly portfolio selection history
-- No foreign key dependencies on other tables; safe to apply after 000018.

-- ============================================================
-- Table: stock_universe
-- Curated list of 50 Vietnamese stocks eligible for weekly scanning
-- ============================================================
CREATE TABLE IF NOT EXISTS "stock-trading".stock_universe (
    ticker          VARCHAR(10) PRIMARY KEY,
    sector          VARCHAR(50) NOT NULL,
    exchange        VARCHAR(10) NOT NULL CHECK (exchange IN ('HOSE', 'HNX', 'UPCOM')),
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    notes           TEXT
);

COMMENT ON TABLE  "stock-trading".stock_universe IS 'Curated universe of 50 Vietnamese stocks eligible for weekly portfolio selection. Volume is computed dynamically from daily_bars.';
COMMENT ON COLUMN "stock-trading".stock_universe.is_active IS 'Only active=TRUE tickers are considered during scanning';

CREATE INDEX IF NOT EXISTS idx_universe_active
    ON "stock-trading".stock_universe(is_active) WHERE is_active = TRUE;

-- ============================================================
-- Table: weekly_portfolio_selection
-- One row per week per candidate stock (selected and near-miss)
-- ============================================================
CREATE TABLE IF NOT EXISTS "stock-trading".weekly_portfolio_selection (
    id               SERIAL PRIMARY KEY,
    week_start       DATE         NOT NULL,   -- Monday date for the trading week
    ticker           VARCHAR(10)  NOT NULL,
    composite_score  NUMERIC(8,6) NOT NULL,
    score_breakdown  JSONB        NOT NULL DEFAULT '{}',  -- per-component scores + p10/p50/p90
    rank             SMALLINT     NOT NULL,   -- 1-5 = selected, 6+ = near-miss
    is_selected      BOOLEAN      NOT NULL DEFAULT FALSE,
    selection_reason TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  "stock-trading".weekly_portfolio_selection IS 'History of weekly portfolio recommendations; is_selected=TRUE for the final 5, FALSE for near-misses';
COMMENT ON COLUMN "stock-trading".weekly_portfolio_selection.week_start IS 'The Monday of the trading week this recommendation is for';
COMMENT ON COLUMN "stock-trading".weekly_portfolio_selection.rank IS '1-5 = final selected portfolio; 6+ = near-misses excluded by sector/correlation constraints';
COMMENT ON COLUMN "stock-trading".weekly_portfolio_selection.score_breakdown IS 'JSONB with return_score, risk_adjusted, liquidity, floor_penalty, momentum components plus raw predictions';

CREATE UNIQUE INDEX IF NOT EXISTS idx_weekly_selection_week_ticker
    ON "stock-trading".weekly_portfolio_selection(week_start, ticker);

CREATE INDEX IF NOT EXISTS idx_weekly_selection_selected
    ON "stock-trading".weekly_portfolio_selection(week_start, is_selected, rank);

-- ============================================================
-- Seed: 50-stock universe with sector and exchange assignments
-- ============================================================
INSERT INTO "stock-trading".stock_universe (ticker, sector, exchange, notes) VALUES
-- Banking
('VCB',  'Banking',       'HOSE', 'Vietcombank - leading state bank'),
('CTG',  'Banking',       'HOSE', 'Vietinbank - state-owned bank'),
('BID',  'Banking',       'HOSE', 'BIDV - major commercial bank'),
('TCB',  'Banking',       'HOSE', 'Techcombank - private bank, growth'),
('VPB',  'Banking',       'HOSE', 'VPBank - high volume'),
('MBB',  'Banking',       'HOSE', 'Military Bank - reliable bank'),
('LPB',  'Banking',       'HNX',  'LPBank - commercial bank'),
('ACB',  'Banking',       'HOSE', 'Asia Commercial Bank'),
('HDB',  'Banking',       'HOSE', 'HDBank - development bank'),
('STB',  'Banking',       'HOSE', 'Sacombank - commercial bank'),
('SHB',  'Banking',       'HNX',  'SHB Bank - commercial bank'),
('TPB',  'Banking',       'HOSE', 'Tien Phong Bank - mid-tier'),
('EIB',  'Banking',       'HOSE', 'Eximbank - export-import bank'),
-- Conglomerate / Real Estate
('VIC',  'Conglomerate',  'HOSE', 'Vingroup - EVs, real estate'),
('VHM',  'Real Estate',   'HOSE', 'Vinhomes - residential developer'),
('VRE',  'Real Estate',   'HOSE', 'Vincom Retail - retail real estate'),
('KDH',  'Real Estate',   'HOSE', 'Khang Dien House'),
('NLG',  'Real Estate',   'HOSE', 'Nam Long - affordable housing'),
('PDR',  'Real Estate',   'HOSE', 'Phat Dat Real Estate'),
('DXG',  'Real Estate',   'HOSE', 'Dat Xanh Group - real estate services'),
-- Technology
('FPT',  'Technology',    'HOSE', 'FPT Corp - IT services'),
-- Energy & Utilities
('GAS',  'Energy',        'HOSE', 'PetroVietnam Gas'),
('POW',  'Energy',        'HOSE', 'PetroVietnam Power'),
('PLX',  'Energy',        'HOSE', 'Petrolimex - oil distribution'),
-- Steel & Industrials
('HPG',  'Steel',         'HOSE', 'Hoa Phat Group - steel producer'),
('REE',  'Industrials',   'HOSE', 'Refrigeration Electrical Engineering'),
('CTD',  'Construction',  'HOSE', 'Coteccons - construction'),
('VCG',  'Construction',  'HOSE', 'Vietnam Construction - infrastructure'),
-- Consumer / Retail
('MWG',  'Consumer',      'HOSE', 'Mobile World - retail'),
('VNM',  'Consumer',      'HOSE', 'Vinamilk - dairy'),
('MSN',  'Consumer',      'HOSE', 'Masan Group - consumer goods'),
('PNJ',  'Consumer',      'HOSE', 'Phu Nhuan Jewelry'),
('SAB',  'Consumer',      'HOSE', 'Sabeco - beverages'),
('KDC',  'Consumer',      'HOSE', 'KIDO Group - food processing'),
-- Aviation
('HVN',  'Aviation',      'HOSE', 'Vietnam Airlines'),
('VJC',  'Aviation',      'HOSE', 'VietJet - low-cost airline'),
-- Securities / Brokerage
('SSI',  'Securities',    'HOSE', 'SSI Securities - brokerage'),
('HCM',  'Securities',    'HOSE', 'HCM Securities'),
('VCI',  'Securities',    'HOSE', 'Vietcap Securities'),
-- Insurance
('BVH',  'Insurance',     'HOSE', 'Bao Viet Holdings'),
-- Logistics
('GMD',  'Logistics',     'HOSE', 'Gemadept'),
('VTP',  'Logistics',     'HOSE', 'Viettel Post'),
-- Commodities / Agriculture
('GVR',  'Commodities',   'HOSE', 'Rubber Group'),
('SBT',  'Agribusiness',  'HOSE', 'Bien Hoa Sugar'),
('ANV',  'Agribusiness',  'HOSE', 'Nam Viet - seafood'),
-- Chemicals & Fertilizers
('DGC',  'Chemicals',     'HOSE', 'Duc Giang Chemicals'),
('DCM',  'Chemicals',     'HOSE', 'Ca Mau Fertilizer'),
('DPM',  'Chemicals',     'HOSE', 'PetroVietnam Fertilizer'),
-- Pharmaceuticals
('DHG',  'Pharma',        'HOSE', 'DHG Pharma'),
-- Industrial Parks
('KBC',  'Industrials',   'HOSE', 'Kinh Bac City - industrial parks')
ON CONFLICT (ticker) DO NOTHING;
