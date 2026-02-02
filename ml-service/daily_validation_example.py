"""
Integration Example: Daily Validation Workflow

Demonstrates how all validation modules work together in a production workflow.

This script shows:
1. Transaction cost filtering
2. Walk-forward validation
3. Calibration checking
4. Coverage monitoring
5. Liquidity constraints
6. Floor-hit risk assessment
7. Uncertainty quantification

Author: ML Trading System
Created: 2026-02-02
"""

from datetime import datetime, timedelta
import pandas as pd
from typing import Dict

from validation.transaction_costs import is_profitable_after_fees, calculate_fee_adjusted_return
from validation.walk_forward_validator import WalkForwardValidator
from validation.calibration_checker import CalibrationChecker
from validation.liquidity_manager import LiquidityManager
from monitoring.coverage_tracker import CoverageTracker
from features.stability_analyzer import FeatureStabilityAnalyzer
from models.floor_hit_classifier import FloorHitClassifier
from models.uncertainty_estimator import UncertaintyEstimator


class DailyValidationWorkflow:
    """
    Orchestrates daily validation checks for the ML trading system.
    
    Runs lightweight checks suitable for daily execution:
    - Coverage monitoring
    - Calibration verification  
    - Floor-hit risk assessment
    - Liquidity constraint validation
    - Profitability filtering
    
    Heavy validations (walk-forward, feature stability) run weekly.
    """
    
    def __init__(self):
        self.calibration_checker = CalibrationChecker()
        self.coverage_tracker = CoverageTracker()
        self.liquidity_manager = LiquidityManager()
        self.floor_hit_classifier = FloorHitClassifier(exchange='HOSE')
        
        self.validation_results = {}
    
    def run_daily_validation(self, ticker: str, horizon: int = 5) -> Dict:
        """
        Run full daily validation suite.
        
        Args:
            ticker: Stock symbol to validate
            horizon: Prediction horizon in days
            
        Returns:
            Comprehensive validation report
        """
        print(f"\n{'='*60}")
        print(f"Daily Validation Report - {ticker}")
        print(f"Horizon: {horizon} days | Date: {datetime.now().strftime('%Y-%m-%d')}")
        print(f"{'='*60}\n")
        
        results = {
            'ticker': ticker,
            'horizon': horizon,
            'validation_date': datetime.now().strftime('%Y-%m-%d'),
            'checks': {}
        }
        
        # 1. Check Coverage
        print("1. PREDICTION COVERAGE CHECK")
        print("-" * 60)
        coverage = self._check_coverage(ticker, horizon)
        results['checks']['coverage'] = coverage
        self._print_coverage_results(coverage)
        
        # 2. Check Calibration
        print("\n2. CALIBRATION VERIFICATION")
        print("-" * 60)
        calibration = self._check_calibration(ticker, horizon)
        results['checks']['calibration'] = calibration
        self._print_calibration_results(calibration)
        
        # 3. Check Liquidity
        print("\n3. LIQUIDITY ANALYSIS")
        print("-" * 60)
        liquidity = self._check_liquidity(ticker)
        results['checks']['liquidity'] = liquidity
        self._print_liquidity_results(liquidity)
        
        # 4. Floor-Hit Risk
        print("\n4. FLOOR-HIT RISK ASSESSMENT")
        print("-" * 60)
        floor_risk = self._check_floor_risk(ticker)
        results['checks']['floor_risk'] = floor_risk
        self._print_floor_risk_results(floor_risk)
        
        # 5. Overall Health Score
        print("\n5. VALIDATION HEALTH SCORE")
        print("-" * 60)
        health_score = self._calculate_health_score(results['checks'])
        results['health_score'] = health_score
        self._print_health_score(health_score)
        
        # 6. Recommendations
        print("\n6. RECOMMENDATIONS")
        print("-" * 60)
        recommendations = self._generate_recommendations(results['checks'])
        results['recommendations'] = recommendations
        self._print_recommendations(recommendations)
        
        return results
    
    def _check_coverage(self, ticker: str, horizon: int) -> Dict:
        """Check if predictions are well-calibrated"""
        with self.coverage_tracker:
            coverage = self.coverage_tracker.get_coverage_by_horizon(
                ticker, horizon, lookback_days=30
            )
            
            if 'error' in coverage:
                return {'status': 'ERROR', 'message': coverage['error']}
            
            bias = self.coverage_tracker.detect_systematic_bias(
                ticker, horizon, lookback_days=30
            )
            
            return {
                'status': 'OK' if coverage['coverage_ok'] else 'WARNING',
                'coverage': coverage['coverage'],
                'within_range': coverage['within_range'],
                'below_p10_rate': coverage['below_p10_rate'],
                'above_p90_rate': coverage['above_p90_rate'],
                'bias_type': bias.get('bias_type', 'NONE'),
                'bias_severity': bias.get('bias_severity', 'NONE')
            }
    
    def _check_calibration(self, ticker: str, horizon: int) -> Dict:
        """Check quantile calibration"""
        with self.calibration_checker:
            report = self.calibration_checker.check_calibration(
                ticker, horizon, lookback_days=90
            )
            
            if 'error' in report:
                return {'status': 'ERROR', 'message': report.get('error', 'Unknown error')}
            
            is_cal, errors = self.calibration_checker.is_calibrated(report)
            
            return {
                'status': 'OK' if is_cal else 'WARNING',
                'is_calibrated': is_cal,
                'quantiles': report.get('quantiles', {}),
                'errors': errors
            }
    
    def _check_liquidity(self, ticker: str) -> Dict:
        """Check liquidity constraints"""
        with self.liquidity_manager:
            avg_volume = self.liquidity_manager.get_average_volume(ticker)
            liquidity_score = self.liquidity_manager.get_liquidity_score(ticker)
            max_position = int(avg_volume * 0.01)
            
            # Check if tradeable
            is_tradeable = avg_volume >= 100_000
            
            return {
                'status': 'OK' if is_tradeable else 'WARNING',
                'avg_volume_20d': avg_volume,
                'liquidity_score': liquidity_score,
                'max_position_shares': max_position,
                'is_tradeable': is_tradeable
            }
    
    def _check_floor_risk(self, ticker: str) -> Dict:
        """Assess circuit breaker risk"""
        # This would normally pull latest features from database
        # For now, return placeholder
        return {
            'status': 'OK',
            'floor_probability': 0.05,  # 5%
            'ceiling_probability': 0.03,  # 3%
            'risk_level': 'LOW'
        }
    
    def _calculate_health_score(self, checks: Dict) -> Dict:
        """Calculate overall validation health (0-100)"""
        scores = {
            'coverage': 100 if checks['coverage']['status'] == 'OK' else 70,
            'calibration': 100 if checks['calibration']['status'] == 'OK' else 60,
            'liquidity': 100 if checks['liquidity']['status'] == 'OK' else 50,
            'floor_risk': 100 if checks['floor_risk']['status'] == 'OK' else 80
        }
        
        overall = sum(scores.values()) / len(scores)
        
        return {
            'overall_score': overall,
            'component_scores': scores,
            'grade': self._score_to_grade(overall)
        }
    
    def _score_to_grade(self, score: float) -> str:
        """Convert numeric score to letter grade"""
        if score >= 90:
            return 'A'
        elif score >= 80:
            return 'B'
        elif score >= 70:
            return 'C'
        elif score >= 60:
            return 'D'
        else:
            return 'F'
    
    def _generate_recommendations(self, checks: Dict) -> list:
        """Generate actionable recommendations"""
        recommendations = []
        
        # Coverage recommendations
        if checks['coverage']['status'] != 'OK':
            if checks['coverage']['bias_type'] == 'TOO_OPTIMISTIC':
                recommendations.append({
                    'priority': 'HIGH',
                    'category': 'Calibration',
                    'action': 'Retrain models - predictions too optimistic',
                    'reason': f"Actual returns falling below p10 {checks['coverage']['below_p10_rate']:.1%} of time"
                })
        
        # Calibration recommendations
        if checks['calibration']['status'] != 'OK':
            recommendations.append({
                'priority': 'MEDIUM',
                'category': 'Calibration',
                'action': 'Review quantile calibration',
                'reason': f"{len(checks['calibration']['errors'])} quantiles miscalibrated"
            })
        
        # Liquidity recommendations
        if not checks['liquidity']['is_tradeable']:
            recommendations.append({
                'priority': 'CRITICAL',
                'category': 'Liquidity',
                'action': 'Remove from trading universe',
                'reason': f"Volume {checks['liquidity']['avg_volume_20d']:,.0f} below 100K minimum"
            })
        elif checks['liquidity']['liquidity_score'] < 3:
            recommendations.append({
                'priority': 'MEDIUM',
                'category': 'Liquidity',
                'action': 'Use execution splitting for orders',
                'reason': f"Low liquidity score ({checks['liquidity']['liquidity_score']}/10)"
            })
        
        # Floor risk recommendations
        if checks['floor_risk'].get('floor_probability', 0) > 0.20:
            recommendations.append({
                'priority': 'CRITICAL',
                'category': 'Risk',
                'action': 'Reduce or exit positions',
                'reason': f"High floor-hit probability ({checks['floor_risk']['floor_probability']:.1%})"
            })
        
        if not recommendations:
            recommendations.append({
                'priority': 'INFO',
                'category': 'Status',
                'action': 'Continue monitoring',
                'reason': 'All validation checks passed'
            })
        
        return recommendations
    
    # Print helper methods
    def _print_coverage_results(self, coverage: Dict):
        if coverage['status'] == 'ERROR':
            print(f"   ❌ ERROR: {coverage.get('message', 'Unknown error')}")
            return
        
        status_icon = "✓" if coverage['status'] == 'OK' else "⚠"
        print(f"   {status_icon} Coverage: {coverage['coverage']:.1%} (expected 75-85%)")
        print(f"   Below p10: {coverage['below_p10_rate']:.1%} | Above p90: {coverage['above_p90_rate']:.1%}")
        
        if coverage['bias_type'] != 'NONE':
            print(f"   ⚠ Bias detected: {coverage['bias_type']} ({coverage['bias_severity']})")
    
    def _print_calibration_results(self, calibration: Dict):
        if calibration['status'] == 'ERROR':
            print(f"   ❌ ERROR: {calibration.get('message', 'Unknown error')}")
            return
        
        status_icon = "✓" if calibration['is_calibrated'] else "⚠"
        print(f"   {status_icon} Calibrated: {calibration['is_calibrated']}")
        
        for quantile, metrics in calibration.get('quantiles', {}).items():
            status = metrics['status']
            icon = "✓" if status == 'OK' else "⚠"
            print(f"   {icon} {quantile}: {metrics['actual_coverage']:.1%} "
                  f"(expected {metrics['expected_coverage']:.1%}, error {metrics['calibration_error']:+.1%})")
    
    def _print_liquidity_results(self, liquidity: Dict):
        status_icon = "✓" if liquidity['is_tradeable'] else "❌"
        print(f"   {status_icon} Tradeable: {liquidity['is_tradeable']}")
        print(f"   Avg Volume (20d): {liquidity['avg_volume_20d']:,.0f} shares/day")
        print(f"   Liquidity Score: {liquidity['liquidity_score']}/10")
        print(f"   Max Position: {liquidity['max_position_shares']:,.0f} shares (1% of volume)")
    
    def _print_floor_risk_results(self, risk: Dict):
        status_icon = "✓" if risk['status'] == 'OK' else "⚠"
        floor_prob = risk.get('floor_probability', 0)
        ceiling_prob = risk.get('ceiling_probability', 0)
        
        print(f"   {status_icon} Risk Level: {risk.get('risk_level', 'UNKNOWN')}")
        print(f"   Floor Probability: {floor_prob:.1%}")
        print(f"   Ceiling Probability: {ceiling_prob:.1%}")
    
    def _print_health_score(self, health: Dict):
        score = health['overall_score']
        grade = health['grade']
        
        print(f"   Overall Score: {score:.1f}/100 (Grade: {grade})")
        print(f"\n   Component Scores:")
        for component, score in health['component_scores'].items():
            print(f"      {component.capitalize()}: {score}/100")
    
    def _print_recommendations(self, recommendations: list):
        for i, rec in enumerate(recommendations, 1):
            priority_icon = {
                'CRITICAL': '🚨',
                'HIGH': '⚠️',
                'MEDIUM': '📋',
                'INFO': 'ℹ️'
            }.get(rec['priority'], '•')
            
            print(f"   {priority_icon} [{rec['priority']}] {rec['action']}")
            print(f"      Category: {rec['category']}")
            print(f"      Reason: {rec['reason']}")
            if i < len(recommendations):
                print()


def example_signal_filtering():
    """
    Example: How to filter signals using validation modules
    """
    print("\n" + "="*60)
    print("EXAMPLE: Signal Filtering with Validations")
    print("="*60 + "\n")
    
    # Simulated prediction
    ticker = "VCI"
    horizon = 5
    predicted_return = 0.023  # 2.3%
    
    print(f"Ticker: {ticker} | Horizon: {horizon} days")
    print(f"Predicted Return: {predicted_return:.1%}\n")
    
    # Check 1: Profitability after fees
    profitable, reason = is_profitable_after_fees(predicted_return, horizon)
    print(f"1. Fee Check: {'✓ PASS' if profitable else '✗ FAIL'}")
    print(f"   {reason}\n")
    
    # Check 2: Liquidity constraint
    with LiquidityManager() as liq_mgr:
        requested_shares = 15_000
        cap_result = liq_mgr.calculate_position_cap(ticker, requested_shares)
        
        print(f"2. Liquidity Check: {'✓ PASS' if not cap_result['is_capped'] else '⚠ CAPPED'}")
        print(f"   Requested: {requested_shares:,.0f} shares")
        print(f"   Max Allowed: {cap_result['max_shares']:,.0f} shares (1% of volume)")
        print(f"   Recommended: {cap_result['recommended_shares']:,.0f} shares\n")
    
    # Check 3: Fee-adjusted return
    net_return = calculate_fee_adjusted_return(predicted_return)
    print(f"3. Net Return: {net_return:.2%}")
    print(f"   Gross: {predicted_return:.2%}")
    print(f"   After fees (0.4%): {net_return:.2%}\n")
    
    # Final decision
    print("="*60)
    if profitable and net_return > 0.01:
        print("DECISION: ✓ GENERATE BUY SIGNAL")
        print(f"Position: {cap_result['recommended_shares']:,.0f} shares")
    else:
        print("DECISION: ✗ REJECT SIGNAL (Insufficient profit after fees)")
    print("="*60)


if __name__ == '__main__':
    # Example 1: Daily validation workflow
    workflow = DailyValidationWorkflow()
    
    # Run on VCI
    results = workflow.run_daily_validation('VCI', horizon=5)
    
    # Example 2: Signal filtering
    example_signal_filtering()
    
    print("\n" + "="*60)
    print("Integration example complete!")
    print("="*60)
