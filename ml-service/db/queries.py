"""
SQL queries for the ML prediction service dynamically bound to the configured schema.
"""
from config import DB_SCHEMA

class Queries:
    @classmethod
    def load_daily_bars(cls):
        return f"""
            SELECT symbol as ticker, date, open, high, low, close, volume, turnover
            FROM "{DB_SCHEMA}".daily_bars
            WHERE symbol = %s
              AND date >= %s
              AND date <= %s
            ORDER BY date ASC
        """

    @classmethod
    def load_daily_bars_recent(cls):
        return f"""
            SELECT symbol as ticker, date, open, high, low, close, volume, turnover
            FROM "{DB_SCHEMA}".daily_bars
            WHERE symbol = %s
            ORDER BY date DESC
            LIMIT %s
        """

    @classmethod
    def get_all_active_tickers(cls):
        return f"""
            SELECT DISTINCT symbol as ticker
            FROM "{DB_SCHEMA}".daily_bars
            WHERE date >= COALESCE(%s::date, (SELECT MAX(date) FROM "{DB_SCHEMA}".daily_bars)) - INTERVAL '7 days'
            ORDER BY symbol
        """

    @classmethod
    def get_all_distinct_dates(cls):
        return f"""
            SELECT DISTINCT date
            FROM "{DB_SCHEMA}".daily_bars
            ORDER BY date ASC
        """

    @classmethod
    def load_features(cls):
        return f"""
            SELECT ticker, date, return_1d, return_5d, return_20d, return_60d,
                   sma_5, sma_10, sma_20, sma_50, sma_200,
                   ema_12, ema_26,
                   rsi_14, rsi_28, macd, macd_signal, macd_hist,
                   bb_upper, bb_middle, bb_lower, bb_width,
                   volume_ratio_5d, volume_ratio_20d, volume_trend, obv,
                   turnover_ratio_5d, turnover_ratio_20d,
                   volatility_5d, volatility_20d, atr_14, coefficient_variation,
                   price_to_sma20, price_to_sma50, price_to_sma200, range_to_close,
                   target_return_5d, target_return_10d,
                   features_complete, feature_version
            FROM "{DB_SCHEMA}".features
            WHERE ticker = %s
              AND date >= %s
              AND date <= %s
              AND features_complete = TRUE
            ORDER BY date ASC
        """

    @classmethod
    def save_features(cls):
        return f"""
            INSERT INTO "{DB_SCHEMA}".features (
                ticker, date,
                return_1d, return_5d, return_20d, return_60d,
                sma_5, sma_10, sma_20, sma_50, sma_200,
                ema_12, ema_26,
                rsi_14, rsi_28, macd, macd_signal, macd_hist,
                bb_upper, bb_middle, bb_lower, bb_width,
                volume_ratio_5d, volume_ratio_20d, volume_trend, obv,
                turnover_ratio_5d, turnover_ratio_20d,
                volatility_5d, volatility_20d, atr_14, coefficient_variation,
                price_to_sma20, price_to_sma50, price_to_sma200, range_to_close,
                target_return_5d, target_return_10d,
                features_complete, feature_version
            ) VALUES (
                %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                %s, %s, %s, %s
            )
            ON CONFLICT (ticker, date)
            DO UPDATE SET
                return_1d = EXCLUDED.return_1d,
                return_5d = EXCLUDED.return_5d,
                return_20d = EXCLUDED.return_20d,
                return_60d = EXCLUDED.return_60d,
                sma_5 = EXCLUDED.sma_5,
                sma_10 = EXCLUDED.sma_10,
                sma_20 = EXCLUDED.sma_20,
                sma_50 = EXCLUDED.sma_50,
                sma_200 = EXCLUDED.sma_200,
                ema_12 = EXCLUDED.ema_12,
                ema_26 = EXCLUDED.ema_26,
                rsi_14 = EXCLUDED.rsi_14,
                rsi_28 = EXCLUDED.rsi_28,
                macd = EXCLUDED.macd,
                macd_signal = EXCLUDED.macd_signal,
                macd_hist = EXCLUDED.macd_hist,
                bb_upper = EXCLUDED.bb_upper,
                bb_middle = EXCLUDED.bb_middle,
                bb_lower = EXCLUDED.bb_lower,
                bb_width = EXCLUDED.bb_width,
                volume_ratio_5d = EXCLUDED.volume_ratio_5d,
                volume_ratio_20d = EXCLUDED.volume_ratio_20d,
                volume_trend = EXCLUDED.volume_trend,
                obv = EXCLUDED.obv,
                turnover_ratio_5d = EXCLUDED.turnover_ratio_5d,
                turnover_ratio_20d = EXCLUDED.turnover_ratio_20d,
                volatility_5d = EXCLUDED.volatility_5d,
                volatility_20d = EXCLUDED.volatility_20d,
                atr_14 = EXCLUDED.atr_14,
                coefficient_variation = EXCLUDED.coefficient_variation,
                price_to_sma20 = EXCLUDED.price_to_sma20,
                price_to_sma50 = EXCLUDED.price_to_sma50,
                price_to_sma200 = EXCLUDED.price_to_sma200,
                range_to_close = EXCLUDED.range_to_close,
                target_return_5d = EXCLUDED.target_return_5d,
                target_return_10d = EXCLUDED.target_return_10d,
                features_complete = EXCLUDED.features_complete,
                feature_version = EXCLUDED.feature_version,
                calculated_at = NOW()
        """

    @classmethod
    def save_prediction(cls):
        return f"""
            INSERT INTO "{DB_SCHEMA}".predictions (
                ticker, prediction_date, target_date, horizon,
                p10, p50, p90, confidence, model_version
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (ticker, prediction_date, target_date, horizon)
            DO UPDATE SET
                p10 = EXCLUDED.p10,
                p50 = EXCLUDED.p50,
                p90 = EXCLUDED.p90,
                confidence = EXCLUDED.confidence,
                model_version = EXCLUDED.model_version,
                updated_at = NOW()
        """

    @classmethod
    def save_model_metadata(cls):
        return f"""
            INSERT INTO "{DB_SCHEMA}".model_metadata (
                model_id, ticker, quantile, horizon,
                training_date, train_start_date, train_end_date, train_days,
                val_start_date, val_end_date,
                hyperparameters, metrics,
                in_production, file_path
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (model_id)
            DO UPDATE SET
                in_production = EXCLUDED.in_production,
                metrics = EXCLUDED.metrics
        """

    @classmethod
    def get_production_models(cls):
        return f"""
            SELECT model_id, ticker, quantile, horizon, file_path, hyperparameters, metrics
            FROM "{DB_SCHEMA}".model_metadata
            WHERE ticker = %s
              AND in_production = TRUE
            ORDER BY horizon ASC, quantile ASC
        """

    @classmethod
    def get_pending_predictions(cls):
        return f"""
            SELECT ticker, prediction_date, target_date, horizon, p10, p50, p90
            FROM "{DB_SCHEMA}".predictions
            WHERE target_date <= %s
              AND actual_return IS NULL
        """

    @classmethod
    def update_prediction_outcome(cls):
        return f"""
            UPDATE "{DB_SCHEMA}".predictions
            SET actual_return = %s,
                error_p10 = %s,
                error_p50 = %s,
                error_p90 = %s,
                updated_at = NOW()
            WHERE ticker = %s
              AND target_date = %s
              AND horizon = %s
        """

    @classmethod
    def set_models_archived(cls):
        return f"""
            UPDATE "{DB_SCHEMA}".model_metadata
            SET in_production = FALSE
            WHERE ticker = %s
        """

    @classmethod
    def get_last_training_date(cls):
        return f"""
            SELECT training_date
            FROM "{DB_SCHEMA}".model_metadata
            WHERE ticker = %s
              AND in_production = TRUE
            ORDER BY training_date DESC
            LIMIT 1
        """

    @classmethod
    def get_latest_prediction_date(cls):
        return f"""
            SELECT MAX(prediction_date)
            FROM "{DB_SCHEMA}".predictions
        """

# Backward compatibility aliases, resolving strings at import time.
LOAD_DAILY_BARS = Queries.load_daily_bars()
LOAD_DAILY_BARS_RECENT = Queries.load_daily_bars_recent()
GET_ALL_ACTIVE_TICKERS = Queries.get_all_active_tickers()
LOAD_FEATURES = Queries.load_features()
SAVE_FEATURES = Queries.save_features()
SAVE_PREDICTION = Queries.save_prediction()
SAVE_MODEL_METADATA = Queries.save_model_metadata()
GET_PRODUCTION_MODELS = Queries.get_production_models()
GET_PENDING_PREDICTIONS = Queries.get_pending_predictions()
UPDATE_PREDICTION_OUTCOME = Queries.update_prediction_outcome()
SET_MODELS_ARCHIVED = Queries.set_models_archived()
GET_LAST_TRAINING_DATE = Queries.get_last_training_date()
GET_ALL_DISTINCT_DATES = Queries.get_all_distinct_dates()
GET_LATEST_PREDICTION_DATE = Queries.get_latest_prediction_date()
