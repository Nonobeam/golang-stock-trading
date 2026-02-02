-- Rollback ML tables

DROP INDEX IF EXISTS "stock-trading".idx_model_quantile;
DROP INDEX IF EXISTS "stock-trading".idx_model_ticker_date;
DROP INDEX IF EXISTS "stock-trading".idx_model_ticker_prod;

DROP INDEX IF EXISTS "stock-trading".idx_predictions_ticker_date;
DROP INDEX IF EXISTS "stock-trading".idx_predictions_pending;
DROP INDEX IF EXISTS "stock-trading".idx_predictions_ticker_target;

DROP INDEX IF EXISTS "stock-trading".idx_features_ticker_recent;
DROP INDEX IF EXISTS "stock-trading".idx_features_complete;
DROP INDEX IF EXISTS "stock-trading".idx_features_ticker_date;

DROP TABLE IF EXISTS "stock-trading".model_metadata;
DROP TABLE IF EXISTS "stock-trading".predictions;
DROP TABLE IF EXISTS "stock-trading".features;
