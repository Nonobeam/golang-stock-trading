\*\*CRITICAL ADJUSTMENTS BASED ON VIETNAMESE MARKET REALITY\*\*



\*\*ACKNOWLEDGMENT OF GAPS\*\*



The three identified architectural gaps are fundamental and must be addressed immediately. The original plan incorrectly assumed instant liquidity similar to US markets.



\*\*REVISED IMPLEMENTATION PLAN\*\*



\*\*GAP 1: T+2 SETTLEMENT REALITY\*\*



\*\*Problem Severity: CRITICAL\*\*



Vietnamese market operates on T+2 settlement:

\- Buy Monday (T+0): shares not yet owned

\- Tuesday (T+1): still in settlement

\- Wednesday (T+2): shares arrive in account

\- Thursday (T+3): can sell



Stop loss logic becomes useless during T+0 to T+2 period.



\*\*Required Changes:\*\*



\*\*Database Schema Addition (Week 1):\*\*



Add to positions table:

\- settlement\_status: enum (LOCKED, LIQUID)

\- purchase\_date: date

\- settlement\_date: date (purchase\_date plus 2 trading days)

\- can\_sell\_date: date (settlement\_date plus 1 day)

\- locked\_capital: decimal (value of shares in settlement)

\- liquid\_capital: decimal (value of tradeable shares)



Add new table position\_settlement\_tracking:

\- position\_id: reference

\- current\_date: date

\- days\_until\_liquid: integer

\- can\_execute\_stop: boolean

\- locked\_value: decimal

\- risk\_classification: enum (HIGH\_RISK\_LOCKED, MODERATE\_RISK\_NEAR\_LIQUID, LOW\_RISK\_LIQUID)



\*\*Risk Calculation Adjustment:\*\*



Two-tier risk calculation:

* Locked shares (T+0 to T+2):
* Cannot execute stop loss: Position is illiquid.
* Risk Calculation (Worst Case Lock):

1. HOSE (7% floor): Calculate risk as 20% of capital invested (entry\_price \* 0.20).
2. HNX (10% floor): Calculate risk as 30% of capital invested (entry\_price \* 0.30).
3. UPCOM (15% floor): Calculate risk as 40% of capital invested (entry\_price \* 0.40).

* Portfolio Impact: This "Locked Risk" value is deducted immediately from your Total Risk Budget.
* Safety Rule: If (Current\_Locked\_Risk + New\_Trade\_Locked\_Risk) > 10% of Total Account Value, REJECT the trade.



Visualizing the T+2.5 "Trap"

To implement the settlement\_status logic correctly in your backend, you need to visualize the state transitions clearly.



Implementation Note: Your Position class needs a scheduled task (Cron Job) that runs every day at 16:30 (after market close).

If T+0 (Purchase Date): Status = LOCKED\_T0

If T+1: Status = LOCKED\_T1

If T+2: Status = LIQUID\_PENDING (Shares arrive after hours)

If T+3: Status = LIQUID (Can Sell)



Liquid shares (T+3 onwards):

\- Can execute stop loss

\- Calculate normal risk: (entry - stop) times shares

\- This is controlled risk



Total portfolio risk:

\- Total\_Risk = Locked\_Risk + Liquid\_Risk

\- Maximum\_Locked\_Risk\_Allowed = 10% of account value

\- Before any new purchase: verify locked risk stays under 10%



\*\*Signal Generation Override:\*\*



Before generating BUY\_NEW or BUY\_MORE:

\- Calculate if purchase would be locked for 3 days

\- Calculate maximum floor-hit loss (7% for HOSE, 10% for HNX)

\- Add this locked risk to current locked risk

\- If total locked risk exceeds 10% account: REJECT purchase regardless of prediction quality

\- Generate message: "Cannot buy: would create excessive locked capital risk"



\*\*Emergency Exit Disclaimer:\*\*



For all stop loss and emergency exit signals:

\- Check settlement status first

\- If LOCKED: generate warning "Stop loss cannot be executed, shares in settlement until \[date]"

\- If LIQUID: proceed with normal stop loss logic

\- Track separately: theoretical stops vs executable stops



\*\*Position Entry Restrictions:\*\*



New rule: 

Monday-Wednesday: 100% Position Size allowed.

Thursday-Friday: 50% Position Size allowed.

Reasoning: If you buy on Friday, you are holding risk for Saturday + Sunday + Monday + Tuesday (settlement). You need to be compensated for that time risk with smaller exposure.



Preferred entry days: Monday, Tuesday

\- Maximum locked period before liquid: 3 trading days

\- Can sell by Thursday/Friday same week



\*\*GAP 2: INTRADAY VS END-OF-DAY DATA\*\*



\*\*Problem Severity: CRITICAL\*\*



Ceiling chase and falling knife are intraday phenomena. Single morning run cannot prevent buying stock that spikes at 10:30 AM.



\*\*Required Solution: OPTION A (Limit Order Strategy)\*\*



Match Vietnamese brokerage reality and morning workflow preference.



\*\*Implementation:\*\*



Pre-market signal generation (7:00-8:30 AM):

\- Run all detection algorithms

\- Generate signals with target entry prices

\- NOT current market price, but limit prices



Limit price calculation:



For BUY\_NEW signals:

\- Calculate reference price: yesterday's close

\- Set maximum entry price: reference times 1.02 (2% above close)

\- Reasoning: if stock gaps up more than 2%, likely ceiling chase

\- Order instruction: "Buy at market, but MAX price = X"



For stocks flagged as ceiling chase risk:

\- Set more conservative limit: reference times 0.98 (2% below close)

\- Wait for pullback within day

\- If never fills, order expires end of day

\- Reasoning: force entry on weakness, not strength



For WAIT\_FOR\_PULLBACK signals:

\- Calculate pullback target: current resistance times 0.95 (5% below)

\- Set limit order at pullback target

\- Good-til-cancelled for 10 days

\- If stock pulls back to target, order fills automatically



\*\*Order Management System:\*\*



New table limit\_orders:

\- order\_id: unique

\- ticker: stock symbol

\- signal\_id: reference to generating signal

\- order\_type: enum (BUY\_LIMIT, SELL\_LIMIT)

\- limit\_price: decimal

\- quantity: integer

\- placed\_date: date

\- expiry\_date: date

\- status: enum (PENDING, FILLED, EXPIRED, CANCELLED)

\- filled\_price: decimal (actual execution)

\- filled\_date: date



Daily order reconciliation (evening):

\- Check which orders filled during day

\- Update position\_tracking if filled

\- Calculate if fill price was favorable vs market close

\- Cancel expired orders

\- Generate new orders for next day



\*\*Ceiling Chase Prevention Through Limits:\*\*



Morning detection:

\- Stock shows ceiling chase pattern from yesterday

\- System wants to generate BUY signal based on predictions

\- BUT ceiling chase flag is active

\- Solution: set limit at yesterday close minus 2%

\- If stock continues up: order never fills (protected)

\- If stock pulls back: order fills at good price (disciplined entry)



\*\*Practical Workflow:\*\*



7:00-8:00 AM: Run full analysis

\- Generate signals

\- Calculate limit prices

\- Create order instructions



8:00-8:30 AM: Review and place orders

\- Manual review of generated orders

\- Place limit orders through broker platform

\- Set quantity and limit prices



9:00-11:30 AM: Market session

\- Orders fill or don't fill based on limit prices

\- No monitoring required

\- System protects through price limits



1:00-3:00 PM: Afternoon session

\- Continued order execution

\- No action needed



Evening: Reconciliation

\- Check which orders filled

\- Update database

\- Prepare for next morning



\*\*Advantages of Limit Order Approach:\*\*



Ceiling chase protection:

\- Stock spikes 7%, your limit at plus 2%: order doesn't fill

\- Automatic protection without real-time monitoring



Falling knife protection:

\- Set limit above current price as stock falling

\- If continues falling, order doesn't fill

\- Protected without real-time monitoring



Execution discipline:

\- Never chase prices

\- Always buy on weakness or at fair value

\- Emotion removed from execution



Compatible with Vietnamese brokers:

\- All brokers support limit orders

\- No special technology needed

\- Works with existing infrastructure



\*\*GAP 3: BACKTESTING MOVED TO WEEK 1\*\*



\*\*Problem Severity: CRITICAL\*\*



Cannot build infrastructure for 7 weeks without validating core algorithms work.



\*\*REVISED WEEK 1: ANALYSIS AND VALIDATION\*\*



\*\*Phase 1A: Historical Data Preparation (Day 1-2)\*\*



Collect required historical data:

\- Daily OHLCV for target stocks: minimum 2 years

\- Calculate all proposed features

\- Identify all floor-hit events historically

\- Identify all ceiling spike events

\- Mark all periods of base formation



Sources:

\- Use existing daily\_bars table

\- Calculate features using existing feature calculator

\- Export to CSV for rapid prototyping



\*\*Phase 1B: Falling Knife Validation (Day 3-4)\*\*



Test falling knife detection algorithm:



Process:

\- Implement falling knife score calculation in Python notebook

\- Run on all historical data

\- For each day with score above 0.7, track what happens next 5 days

\- Calculate: what percentage continued falling

\- Calculate: average additional drop amount

\- Calculate: how often recovered within 10 days



Success criteria:

\- Minimum 75% of detected falling knives continued falling next 5 days

\- Average additional drop exceeds 3%

\- False positive rate under 25%



If fails criteria:

\- Adjust weights in scoring formula

\- Add additional factors (volume, support breaks)

\- Retest until criteria met

\- Document what works and what doesn't



\*\*Phase 1C: Ceiling Chase Validation (Day 5-6)\*\*



Test ceiling chase detection:



Process:

\- Identify all historical single-day spikes above 5%

\- Flag those with volume surge and RSI above 70

\- Track price action next 10 days

\- Calculate: what percentage pulled back 5%+ within 10 days

\- Calculate: if bought at spike, what was average return after 10 days



Success criteria:

\- Minimum 70% of detected ceiling chases pulled back within 10 days

\- Average return if bought at spike: negative or minimal (under 2%)

\- Opportunity cost: waiting for pullback produced better average entry



If fails criteria:

\- Adjust detection thresholds

\- Consider additional factors

\- May discover ceiling chase less relevant for Vietnamese stocks

\- Document findings



\*\*Phase 1D: Base Formation Validation (Day 7-8)\*\*



Test base formation recognition:



Process:

\- Identify all historical consolidation periods meeting criteria

\- Calculate base strength scores

\- Track breakout success after base completion

\- Measure: percentage of bases that led to successful breakouts

\- Measure: average return 10 days after breakout from high-score base



Success criteria:

\- Bases scoring above 0.75 have 65%+ breakout success rate

\- Average return after breakout exceeds 5%

\- Bases reliably predict favorable risk/reward setups



If fails criteria:

\- Adjust base detection criteria

\- May need different approach for Vietnamese market patterns

\- Consider cultural factors (Vietnamese traders may not form Western-style bases)



\*\*Phase 1E: Floor-Hit Prediction Validation (Day 9-10)\*\*



Test floor-hit probability model:



Process:

\- Identify all historical floor-hit events (price hit minus 7% limit)

\- Look at conditions 1-5 days before floor hit

\- Calculate proposed features: momentum, volume surge, distance from support

\- Train simple logistic regression

\- Test prediction accuracy: when model says 30% probability, does floor hit 30% of time?



Success criteria:

\- Model calibrated: predicted probabilities match actual frequencies

\- High probability predictions (above 40%) correctly identify 70%+ of floor hits

\- Low false positive rate: when predicts low probability, rarely hits floor



If fails criteria:

\- Add more predictive features

\- May need more complex model

\- Consider Vietnamese-specific factors (foreign flow, derivative positions)



\*\*Phase 1F: Validation Report and Go/No-Go Decision (Day 11-12)\*\*



Compile findings:



Document for each algorithm:

\- Historical accuracy metrics

\- Confidence intervals

\- Failure modes identified

\- Adjustment recommendations

\- Go or No-Go for building infrastructure



Go/No-Go decision criteria:



GO if:

\- Falling knife detection above 70% accuracy

\- Ceiling chase detection shows clear opportunity cost

\- Base formation predicts favorable setups

\- Floor-hit model better than random (AUC above 0.60)



NO-GO if:

\- Any algorithm fails to beat random chance

\- Vietnamese market patterns don't match assumptions

\- Data quality insufficient for reliable detection



If NO-GO on any component:

\- Do not proceed to Week 2

\- Spend additional 2 weeks refining failed algorithms

\- May need completely different approach

\- Better to fail in Week 1 than discover in Week 8



\*\*REVISED WEEK 2: DATABASE AND INFRASTRUCTURE\*\*



Only proceed after Week 1 validation successful.



Build database schema incorporating all three gaps:

\- T+2 settlement tracking

\- Limit order management

\- Validated algorithm parameters



\*\*ADDITIONAL VIETNAMESE MARKET-SPECIFIC CONSIDERATIONS\*\*



\*\*Floor Lock Detection:\*\*



Add to features table:

\- is\_floor\_locked: boolean

\- floor\_lock\_duration: integer (consecutive days locked)

\- bid\_volume\_at\_floor: integer



Floor lock logic:

\- If price at floor limit AND bid volume zero: flagged as locked

\- Cannot calculate normal metrics (volume patterns break)

\- Automatically disqualify from any buy consideration

\- Generate alert: "Stock floor locked, avoid until recovers"



\*\*ATC Manipulation Detection:\*\*



Vietnamese market has heavy manipulation in final 15 minutes.



New feature calculation:

\- price\_change\_last\_15min: percentage change from 2:45 PM to 3:00 PM close

\- volume\_last\_15min: percentage of daily volume in final 15 minutes

\- atc\_manipulation\_score: weighted combination



Use in floor-hit prediction:

\- If ATC shows heavy selling pressure (drop 3%+ in last 15 min)

\- Increase floor-hit probability by 20 percentage points for next day

\- This is Vietnamese-specific pattern not in Western markets



\*\*Market Trend Override:\*\*



Add to averaging down disqualification (Phase 4A):



Check VN-Index status:

\- If VN-Index below 200-day moving average: market in bear trend

\- If VN-Index dropped 3%+ in single day: market shock

\- Never average down individual stock when market broken

\- Reason: even good stocks fall when market falls



Calculation:

\- Load VN-Index daily data

\- Calculate 50-day and 200-day moving averages

\- Check current index position relative to MAs

\- Flag market regime: BULL, BEAR, NEUTRAL



Averaging override:

\- If market regime BEAR: automatic rejection of all averaging attempts

\- Message: "Market in bearish regime, do not average down any position"



\*\*Liquidity and Spread Considerations:\*\*



Vietnamese small/mid caps have wider spreads than large caps.



Enhanced liquidity scoring:



Tier 1 (VN30 stocks):

\- Tight spreads: 0.1-0.2%

\- Use normal slippage estimate: 0.2%

\- Can trade normal position sizes



Tier 2 (Mid caps):

\- Medium spreads: 0.3-0.5%

\- Increase slippage estimate: 0.5%

\- Reduce position size by 25%



Tier 3 (Small caps):

\- Wide spreads: 1-2%

\- Increase slippage estimate: 1.5%

\- Reduce position size by 50%



Classification:

\- Based on average daily value traded

\- Tier 1: above 100 billion VND daily

\- Tier 2: 20-100 billion VND daily

\- Tier 3: below 20 billion VND daily



Apply in position sizing:

\- Before calculating position size, check liquidity tier

\- Apply tier-based reduction multiplier

\- Adjust expected returns for higher spread costs



\*\*REVISED TIMELINE\*\*



\*\*Week 1: Validation (MUST PASS)\*\*

\- Days 1-2: Data preparation

\- Days 3-4: Falling knife validation

\- Days 5-6: Ceiling chase validation

\- Days 7-8: Base formation validation

\- Days 9-10: Floor-hit validation

\- Days 11-12: Go/No-Go decision



\*\*Week 2: Infrastructure (If Go)\*\*

\- Database schema with T+2 tracking

\- Limit order management tables

\- Settlement status tracking

\- Liquidity tier classification



\*\*Week 3: Core Detection\*\*

\- Implement validated algorithms only

\- Floor lock detection

\- ATC manipulation detection

\- Market regime classification



\*\*Week 4: Signal Logic\*\*

\- Signal generation with limit prices

\- WAIT signals

\- Multi-timeframe confirmation

\- Vietnamese market overrides



\*\*Week 5: Execution and Risk\*\*

\- T+2 risk calculation

\- Locked vs liquid capital tracking

\- Portfolio risk budget with settlement awareness

\- Entry day restrictions (no Thursday/Friday)



\*\*Week 6: Averaging Module\*\*

\- Averaging framework

\- Market trend override

\- Settlement-aware risk calculation

\- Staged protocol



\*\*Week 7: Reporting\*\*

\- Daily workflow with limit order generation

\- Settlement tracking reports

\- Risk dashboards showing locked vs liquid capital

\- Order reconciliation process



\*\*Week 8: Paper Trading\*\*

\- Live operation without real money

\- Validate limit order strategy effectiveness

\- Test settlement tracking accuracy

\- Measure ceiling chase prevention success

\- Collect real market feedback



\*\*SPECIFIC WEEK 2 REFINEMENTS\*\*



\*\*Phase 2A: Falling Knife Detection Enhancement\*\*



Add floor lock handling:

\- Before calculating volume patterns, check if floor locked

\- If locked: set falling knife score to 1.0 (maximum danger)

\- Do not attempt normal calculations (will fail with zero volume)

\- Generate message: "Floor locked, maximum risk"



Special case: consecutive floor locks

\- If locked 2+ consecutive days: extreme danger

\- Probability of extended floor lock period very high

\- Never consider buying regardless of other signals



\*\*SPECIFIC WEEK 3 REFINEMENTS\*\*



\*\*Phase 3C: Floor-Hit Protection with ATC\*\*



Enhanced floor-hit probability:



Base calculation from logistic regression model:

\- Inputs: momentum, volume surge, distance from support, consecutive down days



ATC adjustment:

\- If ATC showed drop above 3% in last 15 minutes yesterday

\- Increase base probability by 0.20 (20 percentage points)

\- Reasoning: demonstrates unloading pressure into close



Final probability:

\- Combined score capped at 1.0

\- Use this enhanced score for all buy/sell decisions



\*\*SPECIFIC WEEK 4 REFINEMENTS\*\*



\*\*Limit Price Strategy\*\*



Standard limit price calculations:



BUY\_NEW in normal conditions:

\- Reference: previous close

\- Limit: reference times 1.02 (allow 2% premium)



BUY\_NEW after ceiling chase detected:

\- Reference: previous close

\- Limit: reference times 0.98 (require 2% discount)



BUY\_MORE for existing position:

\- Reference: previous close

\- Limit: reference times 1.01 (tighter, already have exposure)



WAIT\_FOR\_PULLBACK:

\- Reference: current resistance level identified

\- Limit: reference times 0.95 (5% pullback)



Conservative mode (high market risk):

\- Reduce all limits by additional 2%

\- Require better prices in uncertain conditions



\*\*SPECIFIC WEEK 5 REFINEMENTS\*\*



\*\*T+2 Risk Dashboard\*\*



Daily risk report should show:



Locked Capital Section:

\- Total value in settlement: X VND

\- Percentage of account: Y%

\- Days until liquid: by position

\- Maximum at-risk amount (if all hit floor): Z VND

\- Current unrealized P\&L on locked shares



Liquid Capital Section:

\- Total liquid position value: X VND

\- Controllable risk (entry to stop): Y VND

\- Percentage of account at controllable risk: Z%



Combined Risk Metrics:

\- Total exposure: Locked plus Liquid

\- Uncontrollable risk: Locked at-risk amount

\- Controllable risk: Liquid stop loss risk

\- Risk budget remaining: 6% minus current usage



Alerts:

\- "3 positions settling tomorrow, will be liquid \[date]"

\- "Current locked risk: 8%, approaching 10% limit"

\- "Cannot place new orders until locked risk reduces"



\*\*MEASUREMENT OF SUCCESS\*\*



\*\*Week 1 Validation Success Metrics:\*\*



Must achieve before proceeding:

\- Falling knife detection: 75%+ accuracy

\- Ceiling chase: 70%+ pull back within 10 days

\- Base formation: 65%+ breakout success

\- Floor-hit prediction: AUC above 0.60



\*\*Post-Implementation Success Metrics (3 months):\*\*



T+2 settlement management:

\- Zero stop loss failures due to settlement lock

\- Locked risk never exceeded 10% of account

\- No positions entered on Thursday/Friday (unless manual override)



Limit order effectiveness:

\- 80%+ of ceiling chase patterns: order didn't fill (protected)

\- Average entry prices better than market open by 1%+

\- Zero chase entries at day high



Floor-hit protection:

\- Zero positions that hit floor and got locked

\- All high-probability floor predictions: positions exited before floor

\- No capital trapped in floor-locked stocks



Overall performance:

\- Win rate improved 10+ percentage points

\- Average loss reduced 30%

\- Drawdowns contained under 15%

\- System adherence above 90%



\*\*FINAL IMPLEMENTATION PRIORITY\*\*



\*\*Must Have (Cannot launch without):\*\*

\- Week 1 validation completed successfully

\- T+2 settlement tracking

\- Limit order management

\- Floor lock detection

\- Basic risk budget with settlement awareness



\*\*Should Have (Important but can add incrementally):\*\*

\- ATC manipulation detection

\- Market regime classification

\- Staged averaging protocol

\- Comprehensive reporting dashboard



\*\*Nice to Have (Enhancements over time):\*\*

\- Spring Boot API layer

\- Web dashboard interface

\- Advanced analytics

\- Multi-stock portfolio optimization



\*\*Start simple, validate early, build incrementally.\*\*



The three gaps identified are architectural foundations. Fix these first or entire system will fail in real Vietnamese market conditions.

