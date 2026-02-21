"""
Daily model retrainer module.
Handles checking for model staleness and retraining if necessary.
"""
import sys
import os
import logging
import json
from datetime import datetime, date, timedelta
from typing import Dict, Any

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from models.trainer import ModelTrainer
from db.connection import DatabaseConnection
from db.queries import (
    GET_LAST_TRAINING_DATE, SAVE_MODEL_METADATA, SET_MODELS_ARCHIVED
)
from utils.logging_config import setup_logging

# Configure logging
logger = setup_logging("retrainer")

class Retrainer:
    """Handles model retraining logic."""
    
    def should_retrain(self, ticker: str, days_threshold: int = 30) -> bool:
        """
        Check if models should be retrained.
        
        Args:
            ticker: Stock ticker
            days_threshold: Max days since last training
            
        Returns:
            True if retraining needed
        """
        conn = DatabaseConnection.get_connection()
        try:
            cursor = conn.cursor()
            cursor.execute(GET_LAST_TRAINING_DATE, (ticker,))
            row = cursor.fetchone()
            cursor.close()
            
            if not row:
                logger.info(f"No production models found for {ticker}, retraining needed")
                return True
                
            last_train_date = row[0]
            # Handle date vs datetime
            if isinstance(last_train_date, datetime):
                last_train_date = last_train_date.date()
                
            days_since = (date.today() - last_train_date).days
            
            if days_since >= days_threshold:
                logger.info(f"Models for {ticker} are {days_since} days old, retraining needed")
                return True
            
            logger.info(f"Models for {ticker} are {days_since} days old, valid")
            return False
            
        except Exception as e:
            logger.error(f"Error checking retrain status: {e}")
            # Fail safe: don't retrain on error to avoid storm
            return False
        finally:
            DatabaseConnection.return_connection(conn)

    def retrain_models(self, ticker: str) -> bool:
        """
        Retrain models for ticker and update production status.
        
        Args:
            ticker: Stock ticker
            
        Returns:
            True if successful
        """
        logger.info(f"Starting retraining for {ticker}")
        
        try:
            trainer = ModelTrainer(ticker)
            
            # Train models (use 1000 days or more?)
            # The status file mentioned "trainer.train_all_models(days=1000)"
            results = trainer.train_all_models(days=1000)
            
            # Save results to DB
            self._save_results(ticker, results, trainer)
            
            logger.info(f"Successfully retrained models for {ticker}")
            return True
            
        except Exception as e:
            logger.error(f"Retraining failed for {ticker}: {e}")
            return False

    def _save_results(self, ticker: str, results: Dict[str, Any], trainer: ModelTrainer):
        """Save training results and models."""
        conn = DatabaseConnection.get_connection()
        try:
            cursor = conn.cursor()
            
            # Archive old models first
            cursor.execute(SET_MODELS_ARCHIVED, (ticker,))
            
            # Current date for metadata
            training_date = date.today()
            timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
            
            quantiles = [0.10, 0.50, 0.90]
            
            for q in quantiles:
                q_key = f'p{int(q*100)}'
                if q_key not in results:
                    continue
                    
                model_res = results[q_key]
                model = model_res['model']
                metrics = {'mae': float(model_res['mae'])}
                
                # Model ID and Path
                model_id = f"{ticker}_{q_key}_{timestamp}"
                filename = f"{model_id}.json"
                # Save models in models/saved/TICKER/
                save_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "models", "saved", ticker)
                filepath = os.path.join(save_dir, filename)
                
                # Save execution
                trainer.save_model(model, q, timestamp, filepath)
                
                # Default hyperparams (hardcoded in trainer for now)
                hyperparams = {
                    'quantile_alpha': q,
                    'n_estimators': 200,
                    'max_depth': 5
                }
                
                cursor.execute(SAVE_MODEL_METADATA, (
                    model_id, ticker, q, 1, # Adding explicit horizon=1
                    training_date, 
                    results['train_dates']['start'], results['train_dates']['end'], results['train_dates']['days'],
                    results['val_dates']['start'], results['val_dates']['end'],
                    json.dumps(hyperparams), json.dumps(metrics),
                    True, # in_production
                    filepath
                ))
            
            conn.commit()
            cursor.close()
            
        except Exception as e:
            conn.rollback()
            raise e
        finally:
            DatabaseConnection.return_connection(conn)
