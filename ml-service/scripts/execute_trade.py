#!/usr/bin/env python3
"""
Trade Execution Helper

Records manual trade executions in the positions table.
Updates existing positions or creates new ones based on trade type.

Usage:
    python execute_trade.py buy VCI 170 37500
    python execute_trade.py sell VCI 90 39500
    python execute_trade.py close VCI 35200 stop_loss_triggered
"""

import sys
import os
from datetime import date
import argparse

# Add ml-service to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)) + '/..')

from db.connection import get_connection
from position_manager.manager import PositionManager


def execute_buy(pm: PositionManager, ticker: str, shares: int, price: float, user_id: int = 1):
    """
    Execute buy operation - create new position or add to existing.
    
    Args:
        pm: PositionManager instance
        ticker: Stock symbol
        shares: Number of shares to buy
        price: Execution price per share
        user_id: User ID
    """
    # Check if position already exists
    position = pm.get_position(ticker, user_id)
    
    if position:
        # Add to existing position
        position_id = position['id']
        print(f"Adding {shares} shares to existing {ticker} position...")
        pm.update_position_quantity(position_id, shares, price)
        
        # Retrieve updated position
        updated_pos = pm.get_position(ticker, user_id)
        print(f"✅ Position updated:")
        print(f"   New quantity: {updated_pos['quantity']} shares")
        print(f"   New avg price: {updated_pos['entry_price']:,.2f} VND")
    else:
        # Create new position - prompt for risk parameters
        print(f"Creating new position for {ticker}...")
        print(f"Please provide risk management parameters:")
        
        try:
            stop_loss = float(input(f"Stop loss price (e.g., {price * 0.95:.0f}): "))
            target_1 = float(input(f"Target 1 price (optional, press Enter to skip): ") or 0) or None
            target_2 = float(input(f"Target 2 price (optional, press Enter to skip): ") or 0) or None
            target_3 = float(input(f"Target 3 price (optional, press Enter to skip): ") or 0) or None
            notes = input(f"Notes (optional): ") or f"Manual buy {date.today().isoformat()}"
            
            position_id = pm.add_position(
                user_id=user_id,
                ticker=ticker,
                shares=shares,
                entry_price=price,
                entry_date=date.today().isoformat(),
                stop_loss=stop_loss,
                target_1=target_1,
                target_2=target_2,
                target_3=target_3,
                signal_type='MANUAL',
                notes=notes
            )
            
            print(f"✅ New position created (ID: {position_id})")
            print(f"   Quantity: {shares} shares")
            print(f"   Entry price: {price:,.2f} VND")
            print(f"   Stop loss: {stop_loss:,.2f} VND")
        except KeyboardInterrupt:
            print("\n❌ Trade cancelled")
            return
        except ValueError as e:
            print(f"❌ Invalid input: {e}")
            return


def execute_sell(pm: PositionManager, ticker: str, shares: int, price: float, user_id: int = 1):
    """
    Execute sell operation - reduce position quantity.
    
    Args:
        pm: PositionManager instance
        ticker: Stock symbol
        shares: Number of shares to sell
        price: Execution price per share
        user_id: User ID
    """
    position = pm.get_position(ticker, user_id)
    
    if not position:
        print(f"❌ No active position found for {ticker}")
        return
    
    position_id = position['id']
    current_qty = position['quantity']
    
    if shares > current_qty:
        print(f"❌ Cannot sell {shares} shares, only {current_qty} available")
        return
    
    print(f"Selling {shares} shares of {ticker} @ {price:,.2f}...")
    pm.update_position_quantity(position_id, -shares, price)
    
    # Calculate realized P&L
    avg_price = position['entry_price']
    realized_pnl = shares * (price - avg_price)
    realized_pnl_pct = ((price - avg_price) / avg_price) * 100
    
    print(f"✅ Partial sell executed")
    print(f"   Sold: {shares} shares @ {price:,.2f}")
    print(f"   Realized P&L: {realized_pnl:+,.0f} VND ({realized_pnl_pct:+.2f}%)")
    print(f"   Remaining: {current_qty - shares} shares @ {avg_price:,.2f} avg")


def execute_close(pm: PositionManager, ticker: str, price: float, reason: str, user_id: int = 1):
    """
    Execute position close - sell entire position.
    
    Args:
        pm: PositionManager instance
        ticker: Stock symbol
        price: Exit price per share
        reason: Exit reason (e.g., 'stop_loss_triggered', 'target_3_reached')
        user_id: User ID
    """
    position = pm.get_position(ticker, user_id)
    
    if not position:
        print(f"❌ No active position found for {ticker}")
        return
    
    position_id = position['id']
    quantity = position['quantity']
    avg_price = position['entry_price']
    
    print(f"Closing entire {ticker} position ({quantity} shares) @ {price:,.2f}...")
    pm.close_position(
        position_id=position_id,
        exit_price=price,
        exit_date=date.today().isoformat(),
        exit_reason=reason
    )
    
    # Display P&L (calculated by close_position)
    pnl = quantity * (price - avg_price)
    pnl_pct = ((price - avg_price) / avg_price) * 100
    
    print(f"✅ Position closed")
    print(f"   Closed: {quantity} shares @ {price:,.2f}")
    print(f"   Entry avg: {avg_price:,.2f}")
    print(f"   Total P&L: {pnl:+,.0f} VND ({pnl_pct:+.2f}%)")
    print(f"   Reason: {reason}")


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description='Record manual trade execution',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  Buy 170 shares of VCI at 37,500:
    python execute_trade.py buy VCI 170 37500
  
  Sell 90 shares of VCI at 39,500:
    python execute_trade.py sell VCI 90 39500
  
  Close entire VCI position at 35,200:
    python execute_trade.py close VCI 35200 stop_loss_triggered
        """
    )
    parser.add_argument('action', choices=['buy', 'sell', 'close'], help='Trade action')
    parser.add_argument('ticker', type=str, help='Stock symbol')
    parser.add_argument('arg1', type=str, help='Shares (buy/sell) or Price (close)')
    parser.add_argument('arg2', type=str, nargs='?', help='Price (buy/sell) or Reason (close)')
    parser.add_argument('--user-id', type=int, default=1, help='User ID (default: 1)')
    
    args = parser.parse_args()
    
    try:
        conn = get_connection()
        pm = PositionManager(conn)
        
        if args.action == 'buy':
            if not args.arg2:
                print("❌ Buy requires both shares and price")
                print("Usage: execute_trade.py buy TICKER SHARES PRICE")
                return 1
            shares = int(args.arg1)
            price = float(args.arg2)
            execute_buy(pm, args.ticker, shares, price, args.user_id)
            
        elif args.action == 'sell':
            if not args.arg2:
                print("❌ Sell requires both shares and price")
                print("Usage: execute_trade.py sell TICKER SHARES PRICE")
                return 1
            shares = int(args.arg1)
            price = float(args.arg2)
            execute_sell(pm, args.ticker, shares, price, args.user_id)
            
        elif args.action == 'close':
            if not args.arg2:
                print("❌ Close requires price and reason")
                print("Usage: execute_trade.py close TICKER PRICE REASON")
                return 1
            price = float(args.arg1)
            reason = args.arg2
            execute_close(pm, args.ticker, price, reason, args.user_id)
        
        conn.close()
        return 0
        
    except Exception as e:
        print(f"❌ Error: {str(e)}")
        import traceback
        traceback.print_exc()
        return 1


if __name__ == '__main__':
    sys.exit(main())
