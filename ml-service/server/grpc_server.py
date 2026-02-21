"""
gRPC Server for ML Prediction Service.
"""
import grpc
import logging
import os
import sys
from concurrent import futures
from datetime import datetime
from pathlib import Path

# Add parent directory to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
# Add generated directory to path (for internal imports in generated files)
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "generated"))

from generated import ml_service_pb2
from generated import ml_service_pb2_grpc
from utils.logging_config import setup_logging
from inference.predictor import Predictor
import portfolio.selector as selector
from datetime import date

# Configure logging
logger = setup_logging("grpc_server")

class MLPredictionServicer(ml_service_pb2_grpc.MLPredictionServiceServicer):
    """Implementation of MLPredictionService."""
    
    def __init__(self):
        self.predictor = Predictor()
        logger.info("MLPredictionServicer initialized")

    def Ping(self, request, context):
        """Health check endpoint."""
        logger.debug(f"Received Ping request: {request.message}")
        return ml_service_pb2.PingResponse(
            message="pong",
            server_time=datetime.now().isoformat(),
            version="1.0.0"
        )

    def Predict(self, request, context):
        """Generate prediction for a stock."""
        try:
            ticker = request.ticker
            date = request.date
            logger.info(f"Received Predict request for {ticker} on {date}")
            
            # Use predictor (now returns dict of horizion -> results)
            results = self.predictor.predict_for_date(ticker, date)
            
            # Legacy fields (prefer 1d)
            legacy_p10, legacy_p50, legacy_p90, legacy_conf = 0.0, 0.0, 0.0, 0.0
            
            if 1 in results:
                res = results[1]
                legacy_p10, legacy_p50, legacy_p90, legacy_conf = res['p10'], res['p50'], res['p90'], res['confidence']
            elif len(results) > 0:
                h = sorted(results.keys())[0]
                res = results[h]
                legacy_p10, legacy_p50, legacy_p90, legacy_conf = res['p10'], res['p50'], res['p90'], res['confidence']
            
            # Build list
            predictions_list = []
            for h, res in results.items():
                predictions_list.append(ml_service_pb2.HorizonPrediction(
                    horizon=h,
                    p10=res['p10'],
                    p50=res['p50'],
                    p90=res['p90'],
                    confidence=res['confidence']
                ))
            
            # Get model version for 1d (primary)
            model_version = self.predictor.get_model_version(ticker, 1)
            
            return ml_service_pb2.PredictResponse(
                p10=legacy_p10,
                p50=legacy_p50,
                p90=legacy_p90,
                confidence=legacy_conf,
                model_version=model_version,
                success=True,
                predictions=predictions_list
            )
            
        except Exception as e:
            logger.error(f"Prediction failed: {e}")
            return ml_service_pb2.PredictResponse(
                success=False,
                error_message=str(e)
            )


    def TriggerTraining(self, request, context):
        """
        Automated training pipeline for Telegram bot.
        
        This method:
        1. Checks if training is already in progress (prevents concurrent runs)
        2. Runs backfill_features.py to calculate technical indicators
        3. Runs train.py to train XGBoost models
        4. Returns success/error status
        
        Args:
            request: TriggerTrainingRequest with ticker symbol
            context: gRPC context
            
        Returns:
            TriggerTrainingResponse with success status and model version
        """
        ticker = request.ticker.upper()
        logger.info(f"TriggerTraining called for {ticker}")
        
        # Step 1: Check for concurrent training (simple file-based lock)
        lock_file = Path(f"/tmp/training_{ticker}.lock")
        
        if lock_file.exists():
            logger.warning(f"Training already in progress for {ticker}")
            return ml_service_pb2.TriggerTrainingResponse(
                success=False,
                status="error",
                error_message=f"Training is already in progress for {ticker}. Please wait."
            )
        
        try:
            # Create lock file to prevent concurrent training
            lock_file.touch()
            logger.info(f"Created lock file: {lock_file}")
            
            # Step 2: Backfill features from daily_bars
            logger.info(f"Starting backfill for {ticker}")
            try:
                # Import the backfill function
                sys.path.insert(0, str(Path(__file__).parent.parent))
                from backfill_features import backfill_features
                
                # Run backfill
                backfill_features(ticker)
                logger.info(f"Backfill completed for {ticker}")
                
            except Exception as e:
                logger.error(f"Backfill failed for {ticker}: {e}")
                return ml_service_pb2.TriggerTrainingResponse(
                    success=False,
                    status="error",
                    error_message=f"Feature backfill failed: {str(e)}"
                )
            
            # Step 3: Train the model
            logger.info(f"Starting training for {ticker}")
            try:
                # Import training logic
                from train import main as train_main
                
                # Override sys.argv to pass arguments to train script
                original_argv = sys.argv.copy()
                sys.argv = ['train.py', '--ticker', ticker]
                
                # Run training
                train_main()
                
                # Restore original argv
                sys.argv = original_argv
                
                logger.info(f"Training completed for {ticker}")
                
            except Exception as e:
                logger.exception(f"Training failed for {ticker}")
                sys.argv = original_argv  # Restore even on error
                return ml_service_pb2.TriggerTrainingResponse(
                    success=False,
                    status="error",
                    error_message=f"Model training failed: {str(e)}"
                )
            
            # Step 4: Get model version from database or generate timestamp-based version
            model_version = f"model_{ticker}_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
            
            # Try to get actual version from database
            try:
                from db.connection import get_connection
                conn = get_connection()
                with conn.cursor() as cursor:
                    cursor.execute("""
                        SELECT model_id FROM model_metadata 
                        WHERE ticker = %s AND in_production = TRUE 
                        ORDER BY training_date DESC LIMIT 1
                    """, (ticker,))
                    result = cursor.fetchone()
                    if result:
                        model_version = result[0]
                conn.close()
            except Exception as e:
                logger.warning(f"Could not fetch model version from DB: {e}")
            
            logger.info(f"Successfully trained {ticker}, version: {model_version}")
            
            return ml_service_pb2.TriggerTrainingResponse(
                success=True,
                status="complete",
                model_version=model_version
            )
            
        finally:
            # Always remove lock file, even if training failed
            if lock_file.exists():
                lock_file.unlink()
                logger.info(f"Removed lock file: {lock_file}")

    def TrainModel(self, request, context):
        """Trigger model training."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented yet')
        return ml_service_pb2.TrainModelResponse()

    def GetModelInfo(self, request, context):
        """Get model metadata."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented yet')
        return ml_service_pb2.ModelInfoResponse()

    def RunWeeklyPortfolio(self, request, context):
        """
        Run the weekly portfolio selection pipeline via gRPC.

        If request.pred_date is empty, the most recent prediction date
        available in the database is used (same behaviour as the cron script).

        Report messages are returned in the response — the Go bot sends them
        to Telegram directly, so no Telegram credentials are needed here.
        """
        pred_date = request.pred_date or None

        try:
            if not pred_date:
                from data.loader import DataLoader
                latest = DataLoader.get_latest_prediction_date()
                pred_date = latest if latest else date.today().isoformat()
                logger.info(f"RunWeeklyPortfolio: no date supplied, using {pred_date}")
            else:
                logger.info(f"RunWeeklyPortfolio: ad-hoc run for date {pred_date}")

            # selector.run() builds + would normally self-send Telegram messages.
            # We skip the Telegram send here and return the messages to Go instead.
            messages = selector.run(pred_date=pred_date, skip_telegram=True)
            logger.info(f"RunWeeklyPortfolio: complete. {len(messages)} message(s) to relay.")

            return ml_service_pb2.RunWeeklyPortfolioResponse(
                success=True,
                pred_date=pred_date,
                messages_sent=len(messages),
                report_messages=messages,
            )

        except Exception as e:
            logger.exception(f"RunWeeklyPortfolio failed: {e}")
            return ml_service_pb2.RunWeeklyPortfolioResponse(
                success=False,
                pred_date=pred_date or "",
                error_message=str(e),
            )

def serve(port=None):
    """Start gRPC server."""
    if port is None:
        port = os.getenv('GRPC_PORT', '50051')
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    ml_service_pb2_grpc.add_MLPredictionServiceServicer_to_server(
        MLPredictionServicer(), server
    )
    server.add_insecure_port(f'[::]:{port}')
    logger.info(f"Starting gRPC server on port {port}...")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
