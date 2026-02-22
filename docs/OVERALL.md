# Complete Swing Trading System for Vietnam Stock Market
## A Comprehensive Technical Framework for Weeks-to-Months Timeframe

---

## I. FOUNDATION CONCEPTS

### 1.1 Chart Types and Price Action

**Candlestick Charts (Primary tool)**
Each candlestick represents one time period (daily for your timeframe) and displays:
- **Open**: First traded price of the period
- **High**: Highest price reached during period
- **Low**: Lowest price reached during period
- **Close**: Last traded price of the period

**Candlestick color interpretation:**
- Green/White body: Close > Open (bullish period, buyers dominated)
- Red/Black body: Close < Open (bearish period, sellers dominated)
- Body size: Shows strength of directional movement
- Wick/shadow length: Shows rejection of higher/lower prices

**Key candlestick patterns for swing trading:**
- **Bullish engulfing**: Red candle followed by larger green candle that completely engulfs the previous body - signals potential reversal
- **Hammer**: Small body at top, long lower wick (2-3x body length), minimal upper wick - shows rejection of lower prices
- **Shooting star**: Opposite of hammer - shows rejection of higher prices
- **Doji**: Open and close nearly equal - indecision, often precedes trend change
- **Morning star**: Three candle pattern (down candle, small-bodied candle, up candle) - bullish reversal
- **Evening star**: Opposite of morning star - bearish reversal

**Line Charts**
Connect only closing prices with a line. Useful for:
- Seeing overall trend without noise
- Identifying major support/resistance levels
- Drawing trendlines more clearly

**Volume Bars**
Displayed below price chart, shows number of shares traded in each period.

**Volume interpretation principles:**
- High volume + price up = strong buying, bullish confirmation
- High volume + price down = strong selling, bearish confirmation  
- Low volume + price up = weak move, likely to reverse
- Low volume + price down = selling exhaustion, potential bounce coming
- Volume spike after long consolidation = breakout confirmation
- Volume drying up during pullback in uptrend = healthy consolidation

### 1.2 Multiple Timeframe Analysis

**The timeframe hierarchy for swing trading:**

**Weekly charts (Higher timeframe - context)**
- Determines the major trend direction
- Identifies major support/resistance zones
- Shows market structure (higher highs/higher lows vs lower highs/lower lows)
- Rule: Never trade against the weekly trend for more than quick scalps

**Daily charts (Primary timeframe - decision making)**  
- Where you make entry and exit decisions
- Where you apply most indicators
- Where you identify specific setup patterns
- Where you place your stop losses

**4-hour or Hourly charts (Lower timeframe - execution)**
- Fine-tune entry timing
- Reduce entry price slippage
- Identify micro-support levels for tighter stops
- See intraday momentum shifts

**The synchronization principle:**
You want weekly trend = daily trend = hourly trend all aligned in the same direction. When all three agree, probability of success is highest.

Example of proper alignment:
- Weekly: Stock above 200 SMA, making higher highs
- Daily: Stock above 20 EMA and 50 EMA, both sloping up
- 4-hour: Stock pulling back to 20 EMA on decreasing volume, then bouncing
- Entry: On 4-hour chart when price bounces off support with volume spike

---

## II. TECHNICAL INDICATORS - DETAILED SPECIFICATIONS

### 2.1 Moving Averages (Trend Identification)

**Simple Moving Average (SMA) Calculation:**
```
SMA = (P1 + P2 + P3 + ... + Pn) / n

Where:
P = Price (typically closing price)
n = Number of periods
```

Example for 20-day SMA:
```
Day 1 close: 50,000
Day 2 close: 51,000
...
Day 20 close: 55,000

SMA20 = (50,000 + 51,000 + ... + 55,000) / 20
```

Each day, drop the oldest price and add the newest price.

**Exponential Moving Average (EMA) Calculation:**
```
EMA today = (Price today Ã— K) + (EMA yesterday Ã— (1 - K))

Where:
K = Smoothing factor = 2 / (n + 1)
n = Number of periods

For EMA20: K = 2 / (20 + 1) = 0.0952
```

EMA gives more weight to recent prices, making it more responsive to new price action than SMA.

**Critical moving averages for swing trading:**

**20 EMA (Daily)** - Short-term trend and dynamic support
- Above 20 EMA = short-term uptrend
- Below 20 EMA = short-term downtrend  
- Price bouncing off 20 EMA in uptrend = buy opportunity
- This is your first line of dynamic support

**50 EMA (Daily)** - Intermediate trend
- Above 50 EMA = intermediate uptrend established
- Below 50 EMA = intermediate downtrend
- When 20 EMA crosses above 50 EMA = bullish signal (but lagging)
- When 20 EMA crosses below 50 EMA = bearish signal
- Price pullback to 50 EMA in strong uptrend = second chance entry

**200 SMA (Weekly)** - Long-term trend and major support/resistance
- Above 200 SMA = bull market for that stock
- Below 200 SMA = bear market for that stock
- Crossing above 200 SMA after extended period below = major bullish shift
- This is the "line in the sand" for trend determination

**Moving average slope:**
Not just position matters, but angle of the MA:
- Steep upward slope = strong trend
- Flat MA = ranging market, avoid
- Downward slope = downtrend

**Distance from moving average:**
When price gets too far from its MA (>10-15% for volatile stocks, >5-8% for stable stocks), it tends to revert back. This creates:
- Overbought conditions when too far above
- Oversold conditions when too far below

**IMPORTANT VALIDATION REQUIRED:**
These are starting points. You must backtest on VN30 stocks to find:
- Are 20/50 optimal, or is 15/40 or 25/55 better?
- Should you use EMA or SMA for each?
- Does performance vary by sector?

Test multiple combinations:
- (10, 30) EMAs
- (15, 45) EMAs  
- (20, 50) EMAs
- (25, 60) EMAs
- (20 EMA, 50 SMA)

Record which combination has:
- Highest win rate
- Best risk-adjusted returns
- Lowest false signals

### 2.2 Momentum Oscillators

**RSI (Relative Strength Index) - Complete Calculation**

```
Step 1: Calculate price changes
Change = Today's Close - Yesterday's Close

Step 2: Separate gains and losses
If Change > 0: Gain = Change, Loss = 0
If Change < 0: Gain = 0, Loss = |Change|
If Change = 0: Gain = 0, Loss = 0

Step 3: Calculate average gain and average loss (typically over 14 periods)
First calculation uses simple average:
Average Gain = Sum of Gains over 14 periods / 14
Average Loss = Sum of Losses over 14 periods / 14

Subsequent calculations use smoothed average:
Average Gain = [(Previous Avg Gain Ã— 13) + Current Gain] / 14
Average Loss = [(Previous Avg Loss Ã— 13) + Current Loss] / 14

Step 4: Calculate Relative Strength (RS)
RS = Average Gain / Average Loss

Step 5: Calculate RSI
RSI = 100 - (100 / (1 + RS))
```

**RSI ranges from 0 to 100:**
- 70-100 = Traditionally overbought
- 30-70 = Neutral zone
- 0-30 = Traditionally oversold

**RSI for swing trading - IMPORTANT NUANCES:**

**Traditional interpretation (OFTEN WRONG for trending stocks):**
- Buy when RSI drops to 30 (oversold)
- Sell when RSI rises to 70 (overbought)

**Better interpretation for swing trading:**
In an uptrend:
- RSI pullbacks to 40-50 range = healthy consolidation, buy opportunity
- RSI staying above 40 = trend is strong
- RSI dropping below 40 = trend weakening, be cautious
- RSI above 70 for extended period = very strong trend, don't fade it

In a downtrend:
- RSI rallies to 50-60 range = temporary bounce, sell opportunity  
- RSI staying below 60 = trend is strong downward
- RSI above 60 = trend weakening, downtrend may be ending

**RSI divergence (powerful signal):**
- **Bullish divergence**: Price makes lower low, but RSI makes higher low = momentum shifting bullish, potential reversal
- **Bearish divergence**: Price makes higher high, but RSI makes lower high = momentum weakening, potential reversal

**RSI on multiple timeframes:**
- Weekly RSI for trend strength (>50 = uptrend, <50 = downtrend)
- Daily RSI for entry timing
- Don't rely on RSI alone in strong trends - it will stay "overbought" or "oversold" for long periods

**Validation needed:**
Test RSI with periods of 9, 11, 14, 17, 20 on VN stocks. Find which period gives:
- Fewer false signals
- Better correlation with actual reversals
- Appropriate sensitivity for VN market volatility

**Stochastic Oscillator - Complete Calculation**

```
%K = ((Current Close - Lowest Low in period) / (Highest High in period - Lowest Low in period)) Ã— 100

%D = 3-period SMA of %K (signal line)

Standard settings: (14, 3, 3)
14 = lookback period for high/low
First 3 = smoothing for %K  
Second 3 = smoothing for %D
```

**Stochastic ranges from 0 to 100:**
- 80-100 = Overbought zone
- 20-0 = Oversold zone

**Stochastic for swing trading:**

**In uptrends:**
- Look for %K to dip into 20-40 range (not necessarily below 20)
- Entry signal: %K crosses above %D while both are below 50
- This catches the momentum shift early

**In downtrends:**
- Look for %K to rally into 60-80 range
- Exit/short signal: %K crosses below %D while both are above 50

**Stochastic vs RSI comparison:**
- Stochastic is more sensitive = gives earlier signals but more false signals
- RSI is smoother = gives later signals but more reliable
- Stochastic better for: Range-bound markets, timing entries in consolidations
- RSI better for: Trending markets, confirming trend strength

**The truth:** Neither is "better" - they measure different aspects of momentum. Use both:
- Weekly RSI for trend context
- Daily Stochastic for entry timing
- Require both to agree for highest probability

**Combined rule example:**
Entry only when:
- Weekly RSI > 50 (uptrend confirmed)
- Daily RSI pulled back to 40-55 (healthy consolidation)
- Daily Stochastic crossed up from below 40 (momentum shifting)
- Price above 20 EMA

### 2.3 Trend Strength Indicators

**MACD (Moving Average Convergence Divergence)**

```
MACD Line = 12-period EMA - 26-period EMA
Signal Line = 9-period EMA of MACD Line  
Histogram = MACD Line - Signal Line
```

**MACD interpretation:**

**MACD Line position:**
- Above zero = short-term momentum stronger than intermediate (bullish)
- Below zero = short-term momentum weaker than intermediate (bearish)

**MACD crossovers:**
- MACD crosses above Signal = bullish momentum building
- MACD crosses below Signal = bearish momentum building
- **Warning**: These crossovers lag significantly - price often moves before MACD signals

**Histogram analysis:**
- Histogram growing (bars getting taller) = momentum accelerating
- Histogram shrinking (bars getting shorter) = momentum decelerating
- Zero line cross = actual crossover point

**MACD divergence (more reliable than RSI divergence):**
- Price makes higher high, MACD makes lower high = bearish divergence, trend exhaustion
- Price makes lower low, MACD makes higher low = bullish divergence, reversal potential

**Best use for swing trading:**
Don't use MACD for entries. Use it for:
- Confirming trend changes (crossover + histogram expansion)
- Spotting divergences for potential reversals
- Filtering out weak trends (small histogram = weak trend)

**ADX (Average Directional Index) - Trend Strength**

ADX measures trend strength regardless of direction, from 0-100.

```
Complex calculation involving:
1. True Range (TR) = max of:
   - High - Low
   - |High - Previous Close|
   - |Low - Previous Close|

2. Directional Movement:
   +DM = Current High - Previous High (if positive, else 0)
   -DM = Previous Low - Current Low (if positive, else 0)

3. Directional Indicators:
   +DI = (Smoothed +DM / ATR) Ã— 100
   -DI = (Smoothed -DM / ATR) Ã— 100

4. DX = (|+DI - -DI| / |+DI + -DI|) Ã— 100

5. ADX = Smoothed average of DX over 14 periods
```

**ADX interpretation:**
- ADX < 20 = Weak trend, ranging market - avoid trend-following strategies
- ADX 20-25 = Trend emerging - watch for breakouts
- ADX 25-50 = Strong trend - trend-following strategies work well
- ADX > 50 = Very strong trend - be cautious of exhaustion
- ADX > 75 = Extremely strong trend - often precedes consolidation or reversal

**+DI and -DI (Directional Indicators):**
- +DI > -DI = Uptrend
- -DI > +DI = Downtrend
- Gap between them = trend strength

**Best use for swing trading:**
Use ADX as a filter:
- Only take trend-following trades when ADX > 25
- In range-bound markets (ADX < 20), use mean-reversion strategies instead
- When ADX peaks and starts declining, consider taking profits

**Validation needed for VN market:**
Standard ADX period is 14. Test periods of 10, 12, 14, 16, 18 to find which:
- Best identifies trend vs range conditions
- Gives earliest reliable signals
- Matches VN market characteristics

### 2.4 Volatility Indicators

**Bollinger Bands - Complete Calculation**

```
Middle Band = 20-period SMA of close prices

Standard Deviation calculation:
1. Calculate 20-period SMA (middle band)
2. For each of the 20 periods:
   Deviation = (Price - SMA)Â²
3. Sum all squared deviations
4. Variance = Sum / 20
5. Standard Deviation = âˆšVariance

Upper Band = Middle Band + (2 Ã— Standard Deviation)
Lower Band = Middle Band - (2 Ã— Standard Deviation)
```

**Bollinger Band width:**
```
%B = (Price - Lower Band) / (Upper Band - Lower Band)

%B = 0 means price at lower band
%B = 1 means price at upper band  
%B > 1 means price above upper band
%B < 0 means price below lower band
```

**Bollinger Band interpretation:**

**Band width:**
- Narrow bands (squeeze) = Low volatility, consolidation - often precedes significant move
- Wide bands = High volatility, strong trend
- Bands widening = Volatility increasing, trend accelerating
- Bands narrowing = Volatility decreasing, trend ending

**Price position:**
- Price at upper band in uptrend = strength, not necessarily reversal
- Price at lower band in downtrend = weakness, not necessarily reversal
- Price touching upper band then falling back inside = potential reversal
- Price "walking the band" (staying at upper/lower band) = very strong trend

**Bollinger Bounce strategy (mean reversion):**
- In ranging market (ADX < 20), price bouncing off lower band = buy signal
- Price bouncing off upper band = sell signal
- **Don't use in trending markets** - price can walk the band for extended periods

**Bollinger Breakout strategy (trend following):**
- Squeeze (narrow bands) followed by breakout above upper band on high volume = buy
- First pullback after breakout that holds above middle band = add to position

**Best use for swing trading:**
- Identify consolidation periods (squeeze) for upcoming breakouts
- Gauge normal price range for a stock
- Spot abnormal price extensions (>2 standard deviations)
- Don't use bands as automatic buy/sell levels

**ATR (Average True Range) - Volatility Measurement**

```
True Range = Maximum of:
1. High - Low (current period's range)
2. |High - Previous Close| (gap up scenario)
3. |Low - Previous Close| (gap down scenario)

ATR = Average of True Range over N periods (typically 14)

First ATR = Simple average of first 14 TRs
Subsequent ATRs = [(Prior ATR Ã— 13) + Current TR] / 14
```

**ATR interpretation:**
- Higher ATR = More volatile stock
- Lower ATR = Less volatile stock
- Rising ATR = Volatility increasing
- Falling ATR = Volatility decreasing

**ATR applications for swing trading:**

**1. Stop loss placement:**
```
Long stop = Entry Price - (ATR Ã— Multiplier)
Short stop = Entry Price + (ATR Ã— Multiplier)

Common multipliers: 1.5, 2.0, 2.5, 3.0
```

More volatile stocks need larger multipliers. Less volatile stocks need smaller multipliers.

**2. Position sizing (volatility-adjusted):**
```
Risk per share = ATR Ã— Multiplier
Position size = (Account Risk Amount) / (Risk per share)

Example:
Account: 100,000,000 VND
Risk per trade: 1% = 1,000,000 VND
Stock price: 50,000 VND
ATR: 2,500 VND  
Multiplier: 2

Stop distance = 2,500 Ã— 2 = 5,000 VND
Position size = 1,000,000 / 5,000 = 200 shares
Position value = 200 Ã— 50,000 = 10,000,000 VND (10% of capital)
```

**3. Profit targets:**
```
Target 1 = Entry + (ATR Ã— 2)
Target 2 = Entry + (ATR Ã— 3)
Target 3 = Entry + (ATR Ã— 4)
```

**4. Volatility normalization (ATR%):**
```
ATR% = (ATR / Price) Ã— 100

This normalizes volatility across different price levels.

Example:
Stock A: Price = 100,000, ATR = 5,000, ATR% = 5%
Stock B: Price = 20,000, ATR = 1,000, ATR% = 5%

Both have same relative volatility despite different absolute ATR values.
```

Use ATR% to:
- Compare volatility across stocks of different prices
- Avoid oversized positions in low-priced volatile stocks
- Set consistent stop distances as percentage

**CRITICAL for VN market:**
ATR-based stops assume you can exit at your stop price. 
With daily price limits (Â±7% or Â±10%), if stock gaps to floor, your ATR stop may be unreachable.
HOSE stocks: Â±7% daily limit (most VN30)
HNX stocks: Â±10% daily limit
Ceiling/floor reached = NO MORE TRADING that direction that day

**Solutions:**
1. Use wider ATR multiples (2.5-3x instead of 2x)
2. Reduce position size on volatile stocks near support
3. Never assume you can exit precisely at your stop
4. Consider "hard stop" = accept maximum loss if stock hits floor
5. Size positions small enough that floor-to-floor move (20% total) is acceptable loss

**Validation for VN market:**
Test ATR periods: 10, 12, 14, 16, 20
Test multipliers: 1.5, 2.0, 2.5, 3.0

Find combination that:
- Avoids being stopped out by normal volatility
- Still protects from major adverse moves
- Accounts for gap risk from daily limits

---

## III. VOLUME ANALYSIS

### 3.1 Volume Principles

**Volume basics:**
- Volume confirms price action
- High volume = high conviction
- Low volume = low conviction, suspect the move

**Volume patterns:**

**Healthy uptrend:**
- Volume increases on up days
- Volume decreases on down days (pullbacks)
- This shows buying conviction and weak selling

**Healthy downtrend:**
- Volume increases on down days
- Volume decreases on up days (bounces)

- This shows selling conviction and weak buying

**Warning signs:**
- Price up + volume down = Weak rally, likely to reverse
- Price down + volume down = Weak decline, bounce likely
- Climactic volume at extreme = Potential exhaustion

### 3.2 Volume Indicators

**Volume Moving Average**
```
Volume MA = Average volume over N periods (typically 20 or 50)

Current Volume / Volume MA = Relative volume

Relative volume > 1.5 = Significantly high volume
Relative volume < 0.5 = Significantly low volume
```

**Problem with mean-based volume:**
One or two huge volume spikes distort the average.

**Better approach - Percentile ranking:**
```
Volume percentile = Rank of current volume among last 20 days

Volume > 75th percentile = Top 25% of recent volume days
Volume > 90th percentile = Top 10% of recent volume days
```

Example:
Last 20 days volumes (in millions): [1.2, 1.5, 1.3, 2.8, 1.1, 1.4, 1.3, 1.6, 1.2, 1.5, 1.4, 1.3, 1.2, 1.8, 1.5, 1.4, 1.3, 1.6, 1.2, 2.1]

Sorted: [1.1, 1.2, 1.2, 1.2, 1.2, 1.3, 1.3, 1.3, 1.3, 1.4, 1.4, 1.4, 1.5, 1.5, 1.5, 1.6, 1.6, 1.8, 2.1, 2.8]

75th percentile (top 25%) = 1.6 million
90th percentile (top 10%) = 2.1 million

Current volume of 2.0 million = high but not exceptional
Current volume of 2.5 million = exceptional

**Recommended volume filters:**

For breakouts:
```
Volume > 90th percentile of last 20 days
AND
Volume > median volume Ã— 2
AND  
Absolute volume > minimum liquidity threshold
```

For pullback entries:
```
Volume < 50th percentile during pullback (accumulation)
Then
Volume > 75th percentile on bounce (confirmation)
```

**On-Balance Volume (OBV)**
```
If today's close > yesterday's close: OBV = Previous OBV + Today's Volume
If today's close < yesterday's close: OBV = Previous OBV - Today's Volume
If today's close = yesterday's close: OBV = Previous OBV (unchanged)
```

**OBV interpretation:**
- Rising OBV = Accumulation (buyers absorbing supply)
- Falling OBV = Distribution (sellers overwhelming demand)
- OBV rising while price flat = Building pressure for upward breakout
- OBV falling while price flat = Building pressure for downward break

**OBV divergence:**
- Price making higher highs but OBV not = Bearish divergence, weak rally
- Price making lower lows but OBV not = Bullish divergence, selling exhaustion

**Volume-Weighted Average Price (VWAP)**

For each period:
```
VWAP = Î£(Price Ã— Volume) / Î£(Volume)

Cumulative from market open to current time.
```

**VWAP interpretation:**
- Price above VWAP = Buyers in control for the day
- Price below VWAP = Sellers in control for the day
- Institutional traders often reference VWAP for execution quality

**For swing traders:**
- Less useful than for day traders
- Can use it to judge if your entry price is better or worse than the day's average
- VWAP from previous key days (breakout day, gap day) can act as support/resistance

### 3.3 Liquidity Filters for VN Market

**Minimum liquidity requirements:**

**For VN30 large caps:**
```
Average daily volume > 500,000 shares
AND
Average daily turnover > 2,000,000,000 VND (2 billion VND)
AND
Must have traded every day for last 20 days (no zero-volume days)
AND
Bid-ask spread < 1% (when applicable)
```

**For mid-cap stocks:**
```
Average daily volume > 200,000 shares  
AND
Average daily turnover > 500,000,000 VND (500 million VND)
AND
Must have traded at least 18 of last 20 days
```

**Avoid stocks with:**
- Erratic volume (coefficient of variation > 2)
- Frequent days with zero or minimal volume
- Very wide bid-ask spreads (>2-3%)
- Suspected manipulation patterns (sudden 100x volume spikes unrelated to news)

**Calculate median and mean turnover:**
```
If (Mean turnover / Median turnover) > 2.0:
  Use median for filtering (mean distorted by outliers)
Else:
  Can use mean safely
```

---

## IV. SUPPORT AND RESISTANCE

### 4.1 Identifying Key Levels

**Support** = Price level where buying interest has historically prevented further decline
**Resistance** = Price level where selling pressure has historically prevented further advance

**How to identify:**

**1. Historical turning points:**
Look for prices where:
- Multiple bounces occurred (price touched and reversed at least 2-3 times)
- High volume occurred (indicates significant activity at that level)
- Long wicks or rejection candles formed

**2. Round numbers (psychological levels):**
- 50,000, 100,000, 150,000 VND
- Humans naturally key on round numbers
- Often see clusters of orders at these levels

**3. Previous swing highs and lows:**
- Last significant high before current downtrend = resistance
- Last significant low before current uptrend = support

**4. Gap levels:**
- Unfilled gaps often act as magnetic levels
- Price tends to return to fill gaps

**5. Moving averages as dynamic support/resistance:**
- 20 EMA, 50 EMA, 200 SMA act as moving support/resistance

**Strength of support/resistance depends on:**
- Number of times price tested it (more tests = stronger)
- Volume at the level (higher volume = stronger)
- Time elapsed (older levels are weaker, recent levels stronger)
- Penetration depth (if level was briefly broken then recovered, it's weaker)

**Role reversal principle:**
- Broken resistance becomes new support
- Broken support becomes new resistance
- This is because:
  * Previous buyers trapped above (now resistance)
  * Previous sellers trapped below (now support)

**Example:**
Stock consolidates at 48,000-52,000 for 3 weeks
Breaks above 52,000 on high volume
Pulls back to 52,500 (former resistance, now support)
Bounces from 52,500 = entry signal

### 4.2 Pivot Points

**Standard Pivot Point calculation (from previous period's H/L/C):**

```
Pivot Point (PP) = (High + Low + Close) / 3

Resistance levels:
R1 = (2 Ã— PP) - Low
R2 = PP + (High - Low)  
R3 = High + 2 Ã— (PP - Low)

Support levels:
S1 = (2 Ã— PP) - High
S2 = PP - (High - Low)
S3 = Low - 2 Ã— (High - PP)
```

**For daily pivots:** Use previous day's H/L/C
**For weekly pivots:** Use previous week's H/L/C (more relevant for swing trading)

**Pivot point interpretation:**
- Price above PP = Bullish bias for the period
- Price below PP = Bearish bias for the period
- R1, R2, R3 = Potential resistance/target levels
- S1, S2, S3 = Potential support levels

**Camarilla Pivots (more sensitive):**
```
H4 = Close + ((High - Low) Ã— 1.1 / 2)
H3 = Close + ((High - Low) Ã— 1.1 / 4)
H2 = Close + ((High - Low) Ã— 1.1 / 6)
H1 = Close + ((High - Low) Ã— 1.1 / 12)

L1 = Close - ((High - Low) Ã— 1.1 / 12)
L2 = Close - ((High - Low) Ã— 1.1 / 6)
L3 = Close - ((High - Low) Ã— 1.1 / 4)
L4 = Close - ((High - Low) Ã— 1.1 / 2)
```

**Fibonacci Retracement Levels**

After a significant move from point A (low) to point B (high):

```
Total move = B - A

Retracement levels:
23.6% = B - (Total move Ã— 0.236)
38.2% = B - (Total move Ã— 0.382)
50.0% = B - (Total move Ã— 0.500)
61.8% = B - (Total move Ã— 0.618)
78.6% = B - (Total move Ã— 0.786)
```

Example:
Stock rallies from 40,000 (A) to 60,000 (B)
Total move = 20,000

Fib levels:
23.6%: 60,000 - (20,000 Ã— 0.236) = 55,280
38.2%: 60,000 - (20,000 Ã— 0.382) = 52,360
50.0%: 60,000 - (20,000 Ã— 0.500) = 50,000
61.8%: 60,000 - (20,000 Ã— 0.618) = 47,640

Most pullbacks in healthy uptrends stop at 38.2% or 50% levels.
Pullbacks to 61.8% = deeper correction but still acceptable.
Breaking below 78.6% = trend change likely.

**Fibonacci Extensions (profit targets):**
```
For uptrend from A to B, then pullback to C:

Extension levels from C:
0.618: C + ((B - A) Ã— 0.618)
1.000: C + (B - A)
1.618: C + ((B - A) Ã— 1.618)
2.618: C + ((B - A) Ã— 2.618)
```

These project where the next leg up might reach.

---

## V. ENTRY STRATEGIES - DETAILED PROTOCOLS

### 5.1 Pullback Entry in Established Uptrend

**Prerequisites (all must be true):**
1. Weekly chart shows uptrend (price above 200 SMA, making higher highs/higher lows)
2. Daily chart shows uptrend (price above both 20 EMA and 50 EMA)
3. Both EMAs sloping upward
4. ADX > 25 (trending market, not ranging)
5. Weekly RSI > 50

**The pullback pattern:**
1. Stock rallies away from 20 EMA (up 5-15%)
2. Pulls back toward 20 EMA over 2-7 days
3. Volume decreases during pullback (important - shows selling exhaustion)
4. RSI pulls back to 40-55 range (not oversold, just resting)
5. Price touches or comes within 2% of 20 EMA

**Entry triggers (need at least 2 of these):**
1. Bullish candlestick pattern forms at/near 20 EMA:
   - Hammer
   - Bullish engulfing
   - Morning star
   - Any candle with long lower wick showing rejection

2. Volume spike (above 75th percentile) on the bounce day

3. Stochastic crossover (K crosses above D) in oversold zone

4. Support from previous swing low or Fibonacci level coincides with 20 EMA

5. Price gaps up from the 20 EMA level

**Entry execution:**
```
Entry price = Close of the confirmation candle
Or
Entry price = Previous day's high + 1 tick (breakout entry)
```

**Stop loss placement:**
```
Option 1 (ATR-based):
Stop = Entry - (ATR14 Ã— 2)

Option 2 (Technical):
Stop = Just below the 20 EMA (typically 2-3% below entry)
Or
Stop = Just below the pullback low

Option 3 (Volatility-adjusted):
Stop = Entry - (ATR% Ã— Entry Price)
Where ATR% = (ATR / Price) Ã— 100
```

Choose the method that:
- Gives at least 2:1 reward/risk
- Doesn't exceed 7-8% from entry (for most stocks)
- Accounts for daily limit risk in VN market

**Position sizing:**
```
Risk amount = Account Ã— Risk% (1-2%)
Risk per share = Entry - Stop
Position size = Risk amount / Risk per share

Max position = 20% of account (capital constraint)
Final position = min(Calculated position, Max position)
```

**Profit targets:**
```
Target 1 (T1): Entry + (Entry - Stop) Ã— 2 = 2R
Take 25% off at T1

Target 2 (T2): Entry + (Entry - Stop) Ã— 3 = 3R
Take 25% off at T2

Target 3 (T3): Trail remaining 50% with 20 EMA or 1.5 Ã— ATR
```

**Example trade:**
```
Stock: ABC
Weekly: Above 200 SMA, uptrend
Daily: Above 20/50 EMA, both rising
Price: Rallies to 58,000, pulls back to 20 EMA at 52,000
Volume: Declining during pullback (good)
Entry trigger: Hammer candle + volume spike

Entry: 53,000 (close of hammer)
Stop: 50,000 (below pullback low)
Risk per share: 3,000
Account: 100,000,000 VND
Risk: 1% = 1,000,000 VND
Position: 1,000,000 / 3,000 = 333 shares
Position value: 333 Ã— 53,000 = 17,649,000 VND (17.6% of capital - acceptable)

Targets:
T1 (2R): 53,000 + (3,000 Ã— 2) = 59,000 - Sell 83 shares
T2 (3R): 53,000 + (3,000 Ã— 3) = 62,000 - Sell 83 shares
T3: Trail remaining 167 shares with 20 EMA

Results:
T1 hit: +498,000 VND (0.5% gain on account)
T2 hit: +747,000 VND (0.75% gain on account)
Trailing: Depends on how far it runs
```

### 5.2 Breakout Entry

**Definition:**
Stock consolidates in a range for extended period, then breaks above resistance on strong volume.

**Prerequisites:**
1. Consolidation period: Minimum 20 days (ideally 4-8 weeks)
2. Clear resistance level tested at least 2-3 times
3. Range contraction: Trading range < 1.5 Ã— ATR during consolidation
4. Volume drying up during consolidation (shows supply absorption)
5. Pattern formation: Rectangle, ascending triangle, cup and handle, or flat base

**Identifying valid consolidation:**
```
Range = (Highest High - Lowest Low) during consolidation period
Average price = (Highest High + Lowest Low) / 2
Range % = (Range / Average price) Ã— 100

Valid consolidation: Range% between 8% and 25%
Too tight (< 8%): May lack energy for sustained move
Too wide (> 25%): Not really consolidation, too much volatility
```

**Breakout confirmation criteria (all must be met):**

1. **Price action:**
   - Closes above resistance level
   - Breakout range: At least 0.5 Ã— ATR above resistance (not just 1 tick)
   - No immediate reversal back into range

2. **Volume:**
   - Breakout volume > 90th percentile of consolidation period volume
   - Volume at least 2x median consolidation volume
   - Consistent volume on breakout day (not just last-minute spike)

3. **Context:**
   - Weekly trend aligned (upward)
   - RSI not extremely overbought (< 75)
   - Market (VN-Index) in uptrend or neutral

**False breakout signs (avoid these):**
- Volume on breakout day < median consolidation volume
- Wide-range candle with long upper wick (rejection)
- Breakout at end of trading session (ATC manipulation risk)
- Previous failed breakouts at same level
- Overnight gap without follow-through next day

**Entry methods:**

**Method 1: First pullback entry (preferred for swing trading)**
```
Wait for:
1. Breakout occurs
2. Price rallies 1-3 days
3. Price pulls back to retest breakout level (former resistance, now support)
4. Pullback holds above breakout level (successful retest)
5. Volume dries up on pullback
6. Price bounces on high volume

Entry: On bounce confirmation
Stop: Below breakout level (typically 3-5% below)
```

Advantage: Filters false breakouts, better risk/reward
Disadvantage: May miss explosive moves that don't pull back

**Method 2: Breakout day entry (aggressive)**
```
Entry: Close above resistance OR next day's open
Stop: Below consolidation support or below breakout day low

Only if:
- Volume is extremely high (>90th percentile)
- Strong closing (close in top 25% of day's range)
- Clear pattern (not choppy)
```

**Method 3: Scale-in approach (sophisticated)**
```
Step 1: Enter 50% position on breakout day close
Stop: Below breakout level

Step 2: If pullback occurs and holds above breakout level:
Add remaining 50% on bounce
Adjust stop: Below pullback low

Step 3: If no pullback (runs immediately):
Add remaining 50% on first consolidation above breakout
Adjust stop: Below consolidation
```

**Breakout position sizing:**
```
More conservative than pullback entries because:
- Higher failure rate
- Less defined risk
- Can gap against you

Standard risk: 1% (not 2%)
```

**Example breakout trade:**
```
Stock: XYZ
Consolidation: 45,000 - 48,000 for 6 weeks
Volume: Declining throughout consolidation
Pattern: Ascending triangle

Breakout day:
Open: 47,500
High: 49,200
Low: 47,200
Close: 49,000
Volume: 3.5x median consolidation volume

Analysis:
âœ“ Broke above 48,000 resistance
âœ“ Closed 2.1% above resistance (49,000 vs 48,000)
âœ“ Volume extremely high
âœ“ Strong close (in top 20% of day's range)

Entry approach: Wait for pullback
Days 2-3: Price rallies to 50,500
Day 4-5: Pulls back to 49,200 on low volume
Day 6: Bounces from 49,000 with volume spike

Entry: 49,500
Stop: 47,500 (below breakout level)
Risk: 2,000 per share
Account: 100M VND
Risk amount: 1% = 1M VND
Position: 500 shares = 24.75M VND

Targets:
T1 (2R): 49,500 + 4,000 = 53,500
T2 (3R): 49,500 + 6,000 = 55,500
T3: Trail with 20 EMA
```

### 5.3 Moving Average Crossover Entry

**The setup:**
20 EMA crosses above 50 EMA = Potential trend change from down to up (or range to up)

**Why crossovers lag:**
By the time 20 EMA crosses 50 EMA, price has often already moved significantly. You're late to the party.

**Better approach - Post-crossover pullback entry:**

**Step 1: Identify the crossover**
```
Yesterday: 20 EMA below 50 EMA
Today: 20 EMA above 50 EMA
```

**Step 2: Confirm context**
- Weekly trend is up (or just turning up)
- Price is above both EMAs when cross occurs
- RSI on daily > 50
- ADX > 20 and rising

**Step 3: Wait for pullback**
Don't enter immediately. Wait for:
- Price to pull back to the 20 EMA (or into the zone between 20 and 50 EMA)
- Volume to dry up during pullback
- RSI to pull back to 50-55 range

**Step 4: Entry trigger**
- Price bounces off 20 EMA on high volume
- Bullish candle pattern
- Stochastic crosses up

**Stop placement:**
```
Below the 50 EMA
Or
Below recent swing low
Or
2 Ã— ATR below entry
```

**Position sizing:**
Standard 1-2% risk

**Profit targets:**
```
T1: Previous resistance level or +2R
T2: Next major resistance or +3R
T3: Trail with 20 EMA
```

**Golden Cross variation (longer timeframe):**
```
When 50 SMA crosses above 200 SMA = "Golden Cross"
This is a major trend change signal

Strategy:
- Wait for golden cross on weekly chart
- Then trade pullback entries on daily chart
- Can be aggressive with position sizing (1.5-2% risk)
- Hold longer term (months instead of weeks)
```

### 5.4 Mean Reversion Entry (Range-bound Markets)

**When to use:**
- ADX < 20 (weak trend, ranging market)
- Stock oscillating between clear support and resistance
- Market (VN-Index) in consolidation

**Setup criteria:**
1. Well-defined range (tested 3+ times each side)
2. Range width: 10-20% of price (not too tight, not too wide)
3. Range duration: At least 4 weeks
4. Sufficient volume/liquidity

**Long entry (at support):**
```
Triggers:
- Price touches or slightly penetrates support
- RSI < 30 (or reaches support level's typical RSI)
- Stochastic in oversold zone (<20)
- Volume spike suggesting capitulation
- Bullish reversal candle

Entry: On confirmation candle close
Stop: 3-5% below support
Target: Middle of range, then resistance
```

**Short entry (at resistance)** - if short-selling available:
```
Triggers:
- Price touches resistance
- RSI > 70
- Stochastic overbought (>80)
- Bearish reversal candle

Entry: On confirmation candle close
Stop: 3-5% above resistance
Target: Middle of range, then support
```

**Important limitations:**
- Range trading has lower risk/reward than trend trading
- Ranges eventually break (can trap you)
- Need tight stops
- Position sizing: 1% risk maximum
- Exit immediately if range breaks

### 5.5 Combined Entry Scorecard System

**Create objective scoring for every potential trade:**

**Trend Alignment (3 points max):**
- [ ] +1: Price above 20 EMA
- [ ] +1: Price above 50 EMA
- [ ] +1: Weekly chart in uptrend (above 200 SMA, higher highs/lows)

**Setup Quality (3 points max):**
- [ ] +1: Clear support level present (technical or moving average)
- [ ] +1: Consolidation or pullback pattern visible
- [ ] +1: Volume confirms setup (decreases on pullback, increases on bounce)

**Momentum (2 points max):**
- [ ] +1: RSI in favorable range (40-60 for longs)
- [ ] +1: MACD positive or crossing bullishly + histogram growing

**Risk/Reward (2 points max):**
- [ ] +1: Risk/reward ratio â‰¥ 2:1
- [ ] +1: Stop loss â‰¤ 7% from entry (or â‰¤ 2 Ã— ATR)

**Context (bonus points, can add 0-3):**
- [ ] +1: VN-Index in uptrend (above 50-day MA)
- [ ] +1: Sector showing relative strength
- [ ] +0 to -2: Recent news (positive = +1, neutral = 0, negative = -1 to -2)

**Liquidity (must pass, else score = 0):**
- [ ] Required: Average daily turnover > minimum threshold (2B VND for large caps)
- [ ] Required: No gap days in last 20 days
- [ ] Required: Spread < 2%

**Total possible: 10-13 points**

**Entry rules:**
- Score 7-8: Standard position (1% risk)
- Score 9-10: Larger position (1.5% risk)
- Score 11+: Maximum position (2% risk)
- Score < 7: Pass, do not trade

**Validation:**
After 50-100 trades, analyze:
- Which score ranges had highest win rate?
- Which individual criteria were most predictive?
- Adjust scoring weights based on actual results

Example: If "Volume confirms setup" predicts wins better than anything else, increase its weight to 2 points instead of 1.

---

## 5.6 Vietnam Market Order Execution Protocol

### 5.6.1 Breakout Entry (Vietnam-Adapted)

**Standard GST breakout:** Enter when price closes above resistance on high volume.

**Vietnam problem:** 
- If breakout happens at 9:16 AM and hits ceiling (7% up)
- You cannot enter that day at all
- Must wait for next day
- By next day, might open 7% higher again (second ceiling)
- Chase or skip?

**Vietnam solution:**

**For ceiling-hit breakouts:**
```
Day 1: Stock breaks out, hits ceiling at +7% (e.g., 50K â†’ 53.5K)
âœ— Don't chase on Day 1 (cannot buy at ceiling)

Day 2: Stock opens (check opening price)

Scenario A: Opens at/near ceiling again (53.5K or higher)
â†’ SKIP this trade
â†’ Too much momentum, risk of exhaustion
â†’ Wait for pullback (if it comes)

Scenario B: Opens lower (51K-52K range)
â†’ This is your entry
â†’ Stock pulled back from ceiling but held above breakout
â†’ Place limit buy at 52K
â†’ Stop: Below breakout at 49.5K
â†’ R:R still good (52K entry vs 49.5K stop = 2.5K risk)

Scenario C: Opens back below breakout level (< 50K)
â†’ Failed breakout
â†’ Do not enter
â†’ Wait for new setup
```

**For non-ceiling breakouts:**
```
Stock breaks resistance at 50K, closes at 51.5K (+3%)
âœ“ Can enter on Day 1 close or Day 2 open
âœ“ Use limit order at 51.5-52K
âœ“ Standard GST protocol applies
```

### 5.6.2 Pullback Entry (Vietnam-Adapted)

**GST pullback:** Enter when price bounces off 20 EMA with confirmation.

**Vietnam adaptation:**
```
âœ“ Standard GST approach works well
âœ“ Pullbacks usually don't hit limits
âœ“ Can use continuous trading hours for entry
âœ“ Prefer limit orders at EMA level

Entry checklist:
09:15 - Check if stock in continuous trading (not limit-locked)
09:30 - Place limit buy at 20 EMA price Â± 0.5%
11:30 - If not filled by end morning, reassess
13:00 - Can try again in afternoon if setup still valid
14:15 - Cancel unfilled orders before auction

If limit order doesn't fill:
- Don't chase more than 1% above plan
- If setup is strong, try again next day
- If setup weakens, move to next opportunity
```

### 5.6.3 Stop Loss Placement (Vietnam-Adapted)

**Never use market stop orders in Vietnam.**

**Instead:**

**Method 1: Manual monitoring** (For active traders)
```
- Set price alert at your stop level
- When alert triggers, manually place limit sell order
- Set limit 1-2% below stop (to ensure fill)
- Monitor until filled

Example:
Stop: 47,000
Alert triggers at 47,000
Place limit sell: 46,500 (âˆ’1%)
Likely fills at 46,500-47,000 range
```

**Method 2: Stop-limit orders** (If broker supports)
```
- Stop price: 47,000 (trigger level)
- Limit price: 46,300 (âˆ’1.5% from stop)
- When 47,000 trades, limit sell activates
- Will fill between 46,300-47,000

WARNING: If stock gaps floor (-7%), limit might not fill
This is why position sizing must account for gaps
```

**Method 3: End-of-day stops** (For part-time traders)
```
- Check positions at 14:00 daily
- If any position closed below stop level
- Exit next day at open (market order acceptable)
- Accept overnight gap risk

Only use this if you:
- Size positions very small (0.5% risk)
- Accept potential 10-15% losses
- Cannot monitor intraday
```

**NEVER:**
âŒ Use market stop orders (will get terrible fills)
âŒ Use ATC orders for stops (too unpredictable)
âŒ Assume stop will fill at exact price
âŒ Place stops exactly at limit levels (âˆ’7% = floor, won't fill)

## VI. EXIT STRATEGIES - COMPLETE PROTOCOLS

### 6.1 Profit-Taking Methodology

**The scaling-out framework:**

Never exit entire position at once. Scale out in stages to:
- Lock in some profits (psychological benefit)
- Let winners run (where big money is made)
- Reduce risk progressively
- Adapt to market behavior

**Standard 4-stage exit:**

**Stage 1 - Initial profit (25% of position at +1R):**
```
When: Price reaches entry + 1Ã— risk distance
Action: Sell 25% of shares
Stop adjustment: Move stop on remaining 75% to breakeven (entry price)

Example:
Entry: 50,000
Stop: 47,000 (risk = 3,000)
+1R target: 50,000 + 3,000 = 53,000

At 53,000: Sell 25% of position
New stop on remaining: 50,000 (breakeven)
```

Benefit: Locks in small win, removes entry risk on rest

**Stage 2 - Core profit (25% of position at +2R):**
```
When: Price reaches entry + 2Ã— risk distance
Action: Sell another 25% of shares (25% of original position)
Stop adjustment: Move stop on remaining 50% to +1R (53,000 in example above)

+2R target: 50,000 + 6,000 = 56,000

At 56,000: Sell 25% more
New stop on remaining 50%: 53,000 (+1R)
```

Benefit: Guaranteed profitable trade now, no matter what happens to remainder

**Stage 3 - Major profit (25% of position at +3R or major resistance):**
```
When: Price reaches +3R or hits major technical resistance
Action: Sell another 25%
Stop adjustment: Move stop on remaining 25% to +2R, OR trail with 20 EMA (whichever is higher)

+3R target: 50,000 + 9,000 = 59,000

At 59,000: Sell 25% more
Remaining 25% on trailing stop
```

**Stage 4 - Trail runner (remaining 25%):**
```
Trail with:
Option A: 20 EMA (if still rising)
Option B: 1.5 Ã— ATR trailing stop
Option C: Swing low trailing stop

Example trailing with 20 EMA:
Price at 65,000, 20 EMA at 62,000
Stop: Just below 20 EMA at 61,500

Next day: Price 66,000, 20 EMA moves to 63,000
Stop: 62,500

Exit when price closes below 20 EMA
```

**Alternative scaling framework (for longer holds):**

**3-stage exit:**
- 33% at +2R
- 33% at +3R  
- 34% trail with 50 EMA or major support

**2-stage exit (simplest):**
- 50% at +2R, move stop to breakeven
- 50% trail with 20 EMA

Choose based on:
- Your trading style (more active = more stages)
- Market conditions (choppy = take profits faster)
- Win rate (low win rate = take profits faster)

### 6.2 Stop Loss Management
### 6.2.1 Vietnam-Specific Stop Loss Reality

**The gap risk problem:**
GST assumes you might gap 1-2 stops in worst case.
Vietnam reality: Multi-day limit-down cascades are common.

**Actual observed pattern:**
Day 1: Stop at 49,000, stock hits floor at 46,500 (âˆ’7%) - CANNOT EXIT
Day 2: Opens at floor again: 43,245 (âˆ’7% from yesterday) - CANNOT EXIT
Day 3: Opens at floor: 40,218 (âˆ’7% again) - FINALLY CAN EXIT

Result: Intended 5.8% loss becomes 19.5% loss.

**REVISED STOP STRATEGY:**

**Option 1: "Hard Stop" (Recommended for most traders)**
```
Accept that stops may not fill at intended price.
Size positions assuming WORST CASE = âˆ’15% to âˆ’20% loss

If normal risk = 1% with âˆ’5% stop:
Vietnam risk = 0.3% position size (assuming potential âˆ’15% worst case)
= 1% account risk even if triple-gap occurs

Formula:
Position size = (Account Ã— Risk%) / (Stock price Ã— Worst-case %)
Where Worst-case % = 15% (assumes 2-3 limit downs)

Example:
Account: 100M VND
Risk tolerance: 1% = 1M
Stock: 50,000 VND
Worst case: âˆ’15% = âˆ’7,500 per share

Position = 1M / 7,500 = 133 shares (instead of 333 with normal 3,000 risk)
Position value = 6.65M (6.6% of capital instead of 17.6%)
```

**Option 2: "Trailing Floor" (Aggressive, only for experienced)**
```
Don't use fixed stops at all.
Instead:
- Only enter stocks in strong uptrends (weekly uptrend confirmed)
- Only enter after pullback to support
- Exit only when trend breaks (20 EMA cross, or lower high made)
- Accept that you might hold through volatility

Risk: Can work in bull markets, catastrophic in crashes
Only use if you can handle âˆ’20% drawdowns on positions
```

**Option 3: "Pre-emptive Exit" (Conservative)**
```
Exit BEFORE stop is threatened.

Rules:
- Set stop at âˆ’7% (just above one floor limit)
- If stock drops âˆ’4% to âˆ’5%, exit 50% immediately
- If continues to âˆ’6%, exit remaining 50%
- Never let it actually hit floor

Advantage: Actually get out near intended stop
Disadvantage: Will exit positions that bounce (lower win rate)
```

**RECOMMENDATION FOR GST USERS:**
Use Option 1 (Hard Stop with smaller size) for first 6-12 months.
This protects capital while you learn Vietnam's gap behavior.

**Update all position sizing calculations:**
- Default multiplier for gap risk: 3Ã— intended stop distance
- E.g., if you want to risk 2% with 2Ã—ATR stop:
  - Calculate position for 6Ã—ATR stop
  - This way even triple-gap keeps you at ~2% risk

**Initial stop placement - already covered in entries, but key principles:**

1. **Below technical support:**
   - Swing low
   - Previous resistance-turned-support
   - Moving average
   - Fibonacci level

2. **ATR-based:**
   - 1.5 to 3.0 Ã— ATR below entry
   - Adjust multiplier based on stock volatility
   - Higher volatility = larger multiplier

3. **Percentage-based:**
   - 5-7% for most stocks
   - Adjust based on stock's typical movement

**Stop adjustment rules:**

**Rule 1: Never widen a stop**
If you set stop at 47,000, never move it lower to 45,000.
Only move stops in your favor (up for longs, down for shorts).

**Rule 2: Move to breakeven too early can cause "stops that are too tight"**
Don't automatically move to breakeven at +0.5R or +1R if:
- Volatility is high (ATR% > 5%)
- Stock is consolidating after initial thrust
- No clear support at breakeven level

Better: Move to breakeven +0.5% to +1% above actual entry to account for spread/slippage.

**Rule 3: Time-based stop adjustment**
```
If holding > 3 weeks and position hasn't reached +1R:
- Tighten stop to -0.5R (half original risk)
- Consider exiting if no progress after 4 weeks
```

**Rule 4: Volatility-adjusted trailing**
```
Instead of fixed percentage trail:

Trailing stop = Current High - (Current ATR Ã— Multiplier)

As ATR changes, trailing distance adapts.
More volatile periods = wider trail
Less volatile periods = tighter trail
```

**Rule 5: Support-based trailing** (technical trailing):
```
Identify each higher swing low as price advances.
Place stop just below most recent swing low.

Example:
Entry at 50,000 (initial stop 47,000)
Rallies to 55,000, pulls back to 52,000, bounces
Move stop to 51,500 (below 52,000 swing low)

Rallies to 58,000, pulls back to 54,000, bounces
Move stop to 53,500 (below 54,000 swing low)

Exit when stop is hit or when price makes lower high (trend change)
```

**Rule 6: Time stop (deadweight capital)**
```
If position shows no progress:
< 0.5R profit after:
- 10 days for volatile stocks (ATR% > 5%)
- 15 days for normal stocks (ATR% 3-5%)
- 20 days for stable stocks (ATR% < 3%)
5%)
- 20 days for stable stocks (ATR% < 3%)

Consider exiting if no directional movement.

Exception: If trade thesis still intact and market just consolidating, can hold longer.
```

### 6.3 Emergency Exit Conditions

**Immediate exit (market order, accept slippage) when:**

1. **Thesis invalidation:**
   - Key support level broken decisively (close below, not just intraday wick)
   - Trend reversal confirmed (weekly 20 EMA crosses below 50 EMA)
   - Pattern fails (breakout reverses back into range)

2. **News/event risk:**
   - Negative earnings surprise
   - Regulatory investigation announced
   - Management scandal
   - Sector-wide negative news

3. **Technical deterioration:**
   - Volume spike with price decline (distribution)
   - Gap down on high volume
   - Consecutive limit-down days
   - Topping pattern completion (head and shoulders, double top confirmed)

4. **Portfolio risk:**
   - Daily loss limit hit (see portfolio rules)
   - Correlation spike (all positions moving against you simultaneously)
   - Margin call risk (if using margin)

5. **Market environment shift:**
   - VN-Index breaks major support
   - Market volatility spike (VIX equivalent for VN spikes >50%)
   - Liquidity crisis signs

**Partial emergency exit (50-75% of position):**
- Unexpected high-volume reversal day
- Market flash crash or extreme volatility
- Geopolitical events affecting VN market
- Your own analysis error discovered

### 6.4 Profit Target Calculation Methods

**Method 1: R-multiple targets (already covered)**
```
T1 = Entry + (Entry - Stop) Ã— 2
T2 = Entry + (Entry - Stop) Ã— 3
T3 = Entry + (Entry - Stop) Ã— 4+
```

**Method 2: Technical resistance targets**
```
Identify:
- Previous swing highs
- Round numbers (50,000, 100,000, etc.)
- Previous consolidation zones
- Fibonacci extension levels

Set targets just below these levels to improve fill probability.

Example:
Previous swing high: 58,500
Set target: 58,000 (allows room for multiple sellers at round number)
```

**Method 3: Measured move targets**
```
For breakouts:
Breakout target = Resistance + (Resistance - Support of consolidation)

Example:
Consolidation range: 45,000 (support) to 50,000 (resistance)
Range size: 5,000
Breakout target: 50,000 + 5,000 = 55,000

For pullbacks:
If prior leg up was 15%, expect similar percentage on next leg.
```

**Method 4: ATR-based targets**
```
T1 = Entry + (2 Ã— ATR)
T2 = Entry + (3 Ã— ATR)
T3 = Entry + (4 Ã— ATR)

Advantage: Adapts to stock's volatility
More volatile stocks = larger absolute targets
Less volatile stocks = smaller absolute targets
```

**Method 5: Time-based profit taking**
```
If holding > normal duration for your system:
- Average hold time for winners = 18 days
- At day 25, consider taking 50% profit regardless of price
- At day 35, exit entire position

Rationale: Redeployment of capital may offer better opportunities
```

**Hybrid approach (recommended):**
```
T1 = Closer of (2R or next technical resistance)
T2 = Closer of (3R or major resistance)
T3 = Further of (4R or measured move target), then trail
```

---

## VII. RISK MANAGEMENT - COMPLETE FRAMEWORK

### 7.1 Position Sizing - Detailed Formulas

**Basic position sizing (fixed risk per trade):**

```
Position Size = (Account Risk Amount) / (Risk Per Share)

Where:
Account Risk Amount = Total Capital Ã— Risk Percentage (1-2%)
Risk Per Share = Entry Price - Stop Loss Price

Example:
Capital: 100,000,000 VND
Risk per trade: 1.5% = 1,500,000 VND
Entry: 52,000
Stop: 49,000
Risk per share: 3,000 VND

Position = 1,500,000 / 3,000 = 500 shares
Position value = 500 Ã— 52,000 = 26,000,000 VND (26% of capital)
```

**Capital constraint check:**
```
If (Position Value / Total Capital) > Max Position % (typically 20-25%):
    Position = (Total Capital Ã— Max Position %) / Entry Price
    Recalculate actual risk
```

Example continued:
26% > 20% maximum
Adjusted position = (100,000,000 Ã— 0.20) / 52,000 = 384 shares
Actual risk = 384 Ã— 3,000 = 1,152,000 VND (1.15% of capital)

**Volatility-adjusted position sizing:**

```
ATR% = (ATR / Price) Ã— 100

Volatility adjustment factor:
If ATR% < 3%: Multiply position by 1.2 (low volatility, can be more aggressive)
If ATR% 3-5%: No adjustment (normal volatility)
If ATR% 5-8%: Multiply position by 0.8 (high volatility, reduce size)
If ATR% > 8%: Multiply position by 0.6 (extreme volatility, significantly reduce)

Adjusted Position = Base Position Ã— Volatility Factor
```

Example:
Base position: 500 shares
ATR = 3,500
Price = 52,000
ATR% = (3,500 / 52,000) Ã— 100 = 6.7% (high volatility)

Adjusted position = 500 Ã— 0.8 = 400 shares
Position value = 400 Ã— 52,000 = 20,800,000 VND
Risk = 400 Ã— 3,000 = 1,200,000 VND (1.2% of capital)

**Score-based position sizing (from scorecard system):**

```
If Trade Score = 7-8: Risk 1.0% of capital (standard)
If Trade Score = 9-10: Risk 1.5% of capital (high conviction)
If Trade Score = 11+: Risk 2.0% of capital (maximum)

Then apply volatility adjustment on top.
```

**Correlation-adjusted position sizing:**

```
For each new position, calculate correlation with existing positions:

Correlation coefficient (Ï) between -1 and +1:
Ï > 0.7 = High positive correlation (move together)
Ï 0.3 to 0.7 = Moderate correlation
Ï -0.3 to 0.3 = Low correlation  
Ï < -0.3 = Negative correlation (hedge)

Adjustment:
If adding position with Ï > 0.7 to any existing position:
    New position size = Base size Ã— 0.5

If adding position with Ï > 0.85:
    Don't add (too correlated)

If total weighted correlation of portfolio > 0.6:
    Reduce all position sizes by 20%
```

To calculate correlation:
```
Use 30-60 days of daily returns for each stock
Ï = Covariance(Stock A returns, Stock B returns) / (StdDev(A) Ã— StdDev(B))

Or use available tools/libraries to calculate.
```

**Gap risk adjustment (VN-specific):**

```
If stock has history of gap moves:
- Count gaps > 5% in last 60 days
- If > 3 gaps: Reduce position by 30%
- If > 5 gaps: Reduce position by 50%

If entering position near daily limit:
- If within 3% of ceiling/floor: Reduce position by 40%
- Risk of being locked in at limit for multiple days
```

### 7.2 Portfolio-Level Risk Rules

**Aggregate risk limits:**

```
Maximum total portfolio risk at any time:
Sum of (Position Size Ã— Risk per share) across all open positions â‰¤ 6% of capital

Example:
Capital: 100,000,000 VND
Max aggregate risk: 6,000,000 VND

Current positions:
Position 1: 500 shares Ã— 3,000 risk = 1,500,000
Position 2: 300 shares Ã— 4,000 risk = 1,200,000
Position 3: 600 shares Ã— 2,500 risk = 1,500,000
Total risk: 4,200,000 VND (4.2% - can add more)

Can add new position risking up to: 6,000,000 - 4,200,000 = 1,800,000
```

**Position concentration limits:**

```
Maximum positions:
- In same sector: 3 positions
- Total open positions: 6-8 positions maximum

Reasoning:
- Too many positions = cannot monitor adequately
- Too concentrated = single event can destroy portfolio
- 6-8 positions = sweet spot for swing trading
```

**Sector exposure limits:**

```
Maximum capital in single sector: 40% of portfolio

Example:
100M capital, maximum 40M in banking stocks combined

VN sectors to track:
- Banking & Finance
- Real Estate  
- Manufacturing
- Resources (Steel, Mining)
- Retail & Services
- Technology
- Energy
```

**Correlation limits:**

```
Calculate portfolio-wide correlation matrix.

Weighted average correlation:
If > 0.6: Portfolio too correlated, reduce position sizes
If > 0.75: Stop adding correlated positions

During market stress, correlations spike toward 1.0 (everything moves together).
Monitor and adjust.
```

### 7.3 Drawdown Protection Rules

**Daily loss limits:**

```
If daily loss reaches -2% of capital:
- Stop taking new positions for the day
- Review existing positions
- Only exit positions or tighten stops
- No revenge trading

If daily loss reaches -3% of capital:
- Close all positions
- Stop trading for remainder of day
- Review what went wrong
```

**Weekly loss limits:**

```
If weekly loss reaches -5% of capital:
- Reduce position sizes by 50% for next week
- Review trading system
- Check if market regime changed

If weekly loss reaches -7% of capital:
- Stop trading for remainder of week
- Comprehensive system review
- Paper trade only next week
```

**Monthly loss limits:**

```
If monthly loss reaches -10% of capital:
- Stop live trading
- Paper trade for 2 weeks minimum
- Review last 30 trades
- Identify systematic errors
- Revise system if needed

If monthly loss reaches -15% of capital:
- Stop trading for remainder of month
- Complete system overhaul
- May indicate market regime incompatible with system
```

**Maximum drawdown limit (absolute):**

```
If drawdown from peak equity reaches -20%:
- Stop all trading
- Capital preservation mode
- Complete psychological and system reset
- May need extended break (1-3 months)
- Restart with smaller capital allocation
```

**Consecutive loss rules:**

```
After 3 consecutive losses:
- Reduce next position size by 30%
- Review those 3 trades for common errors

After 5 consecutive losses:
- Reduce position sizes by 50%
- Take 3-5 day break
- Paper trade to regain confidence

After 7 consecutive losses:
- Stop trading
- System clearly not working in current environment
- Need significant adjustments
```

### 7.4 Win Streak Management

**Overconfidence protection:**

```
After 3 consecutive wins:
- Maintain standard position sizing (don't increase)
- Avoid overtrading
- Stick to system (don't get creative)

After 5+ consecutive wins:
- Take 1 day break
- Review if market conditions are exceptionally favorable
- Consider taking some profits off table
- Don't assume you've "figured it out"

After exceptionally large win (>5R):
- Take partial profits immediately
- Don't increase position sizes on next trades
- Euphoria is dangerous - stay mechanical
```

**Equity curve management:**

```
Track high-water mark (peak account equity).

When at new high-water mark:
- Can use standard position sizing (1-2% risk)

When in drawdown from high-water mark:
Scale down based on drawdown:
- 0-5% drawdown: Normal sizing (1-2%)
- 5-10% drawdown: Reduce to 1% risk per trade
- 10-15% drawdown: Reduce to 0.5% risk per trade
- >15% drawdown: Stop trading (as per monthly limits above)
```

### 7.5 Event Risk Management

**Scheduled events (avoid or adjust):**

```
Don't enter new positions or significantly increase risk when:
- Earnings report within 3-5 days
- Ex-dividend date within 3 days (price adjustment)
- Annual General Meeting scheduled
- Known regulatory announcements
- Major economic data releases (GDP, CPI, policy rate)
- Public holidays approaching (liquidity drops)

If already in position:
- Consider taking partial profits
- Tighten stops
- Reduce size before event
```

**Unscheduled events (reaction protocol):**

```
Company-specific news:
Negative: Exit immediately if material
Positive: Hold but prepare to take profits into strength

Sector news:
Affects multiple holdings: Assess correlation, may need to exit several

Market-wide news:
VN-Index gaps down >3%: Close all positions or hold only strongest
Geopolitical crisis: Reduce exposure by 50%+
```

---

## VIII. VIETNAM MARKET SPECIFIC CONSIDERATIONS

### 8.1 Daily Price Limits

"CRITICAL: Unlike markets with circuit breakers that pause trading, Vietnam's limits LOCK prices. If a stock hits ceiling at 9:30 AM, it stays there all day - you cannot buy above that price until tomorrow. This makes breakout entries on limit-up days impossible to execute."

**Understanding limit structure:**

Most stocks: Â±7% daily limit (Â±10% for some)
```
Reference price (yesterday's close): 50,000
Ceiling price: 50,000 Ã— 1.07 = 53,500
Floor price: 50,000 Ã— 0.93 = 46,500

Stock can only trade between 46,500 and 53,500.
```

**Implications for trading:**

**1. Gap risk:**
Stock can gap from floor to floor or ceiling to ceiling across multiple days.
Your stop loss may be unreachable.

Example:
Entry: 52,000
Stop: 49,000

Day 1: Stock hits floor at 48,360 (you cannot exit)
Day 2: Opens at floor again: 45,010 (still cannot exit)
Day 3: Opens at floor: 41,860

Actual exit: 41,860 vs intended stop: 49,000
Loss: 19.5% instead of 5.8%

**Solution:**
```
Position sizing with gap risk factor:

Worst case scenario = Multiple floor hits

If normal position with 1% risk:
With gap risk, reduce position to 0.5-0.7% risk

This ensures that even worst-case floor scenario doesn't exceed 3-4% loss.

Or:

Set mental stop wider than ATR stop:
Technical stop: 49,000 (5.8%)
Gap-adjusted mental stop: Accept up to 15% loss
Position size: Risk only 0.5% using the 15% potential loss
```

**2. Ceiling/floor price patterns:**

```
Multiple ceiling days = Extreme strength (often 3-5+ days)
Buying pressure overwhelming

Don't chase on first ceiling day.
Wait for:
- First day price opens below ceiling
- Consolidation pattern
- Lower risk entry

Multiple floor days = Extreme weakness  
Selling pressure overwhelming

Don't try to catch falling knife.
Wait for:
- Stabilization (no longer hitting floor)
- Base building
- Reversal confirmation
```

**3. Auction dynamics:**

```
Opening auction (ATO): 9:00-9:15
Can get wide spreads and gaps from previous close

Continuous trading: 9:15-11:30 (morning), 13:00-14:30 (afternoon)

Closing auction (ATC): 14:30-14:45

ATC often sees manipulation:
- Last-minute orders to move price
- Can create false breakouts/breakdowns
- Be skeptical of close-only moves

Strategy:
Avoid placing stops that execute at auction prices.
Use limit orders during auctions.
```

### 8.2 Settlement Rules (TRANSITIONING)

**Traditional system (still dominant as of late 2025):**
- T+2 settlement for most stocks
- Cash locks for 2 days after purchase
- Cannot use unsettled funds for new trades

**New KRX platform (launched 2025, limited adoption):**
- T+0 settlement available
- Same-day trading possible
- Short-selling enabled (on limited stocks)
- NOT YET widely adopted - most retail still uses T+2

**Practical implications TODAY:**
- Assume T+2 unless confirmed your broker offers T+0
- Keep 20-30% cash buffer (not just for opportunities, but for settlement lag)
- Don't assume you can flip positions same-day
- When calculating position sizes, account for capital being tied up 2+ days

**Example of settlement trap:**
Monday: Sell Position A (50M VND) - cash available Wednesday
Monday: Want to buy Position B (50M VND) - CANNOT if you only have 50M total
Solution: Must have had 100M to do both, or wait until Wednesday

**As T+0 adoption grows:**
Monitor your broker's platform upgrades. When T+0 becomes available:
- Can reduce cash buffer to 10-15%
- Can implement faster rotation strategies
- Short-selling becomes viable (major system addition)

### 8.3 Trading Session Patterns

### 8.3.1 Auction Execution Rules (CRITICAL for Vietnam)

**Opening Auction (ATO) 9:00-9:15:**
- Orders accumulate but DON'T execute until 9:15
- Clearing price determined by supply/demand balance
- If ONLY ATO orders exist with no match, NO TRADE occurs
- Market orders (MP) in auction get filled at clearing price
- Limit orders only fill if clearing price meets/betters limit

**Implications for entries:**
DON'T use ATO for momentum breakouts - price is unpredictable
DON'T assume you'll get "opening price" - might not fill at all
DO wait until 9:15 continuous trading starts for breakout entries
DO use limit orders with specific price if entering in auction

**Closing Auction (ATC) 14:30-14:45:**
- Similar mechanics to opening
- High manipulation risk (last-minute large orders can swing price)
- Closing price can deviate significantly from 14:30 price

**Strategy adjustments:**
- For breakout entries: Wait until 9:20-9:30 (after auction clears) to see real momentum
- For stop losses: Use continuous trading hours (9:15-11:30, 13:00-14:30)
- Avoid placing critical orders during auctions unless absolutely necessary

**Morning session (9:15-11:30):**
- Highest volume (typically 50-60% of daily volume)
- Most volatility
- Best liquidity for entries/exits
- Institutional activity concentrated here
- Best time for breakout entries

**Lunch break (11:30-13:00):**
- No trading
- Time to review positions and plan

**Afternoon session (13:00-14:30):**
- Lower volume (30-40% of daily)
- Often choppy, range-bound
- Individual investors more active
- May see profit-taking
- Less reliable for technical signals

**Closing auction (14:30-14:45):**
- 10-15% of daily volume
- Can see manipulation
- Price often reverts after auction
- Avoid using closing price alone for signals

**Strategy by session:**

```
Entries: Prefer morning session
- Better liquidity
- Clearer price discovery
- Can establish position with good fill

Exits: Monitor both sessions
- Use limit orders in afternoon if less urgent
- Use market orders in morning if need quick exit

Stop management:
- Don't let stops trigger at auction
- Use GTC limit orders for stops, not market orders
```

### 8.4 Liquidity: The Make-or-Break Factor

**CRITICAL REALITY CHECK:**
HOSE has ~400 listings, but practical swing trading universe is ~30-50 stocks maximum.

**VN30 stocks (30 largest):**
- Account for majority of daily volume (1,422M shares avg)
- Narrow spreads (<1%)
- Can execute multi-lot orders reliably
- **THESE SHOULD BE 80-90% OF YOUR TRADES**

**Mid-cap stocks (next 50-100):**
- Tradeable but require smaller positions
- Expect 1-2% spreads
- May not fill entire order in one day
- **Use maximum 20% of your capital here**

**Small caps (rest of market):**
- AVOID for systematic swing trading
- Too illiquid for reliable execution
- High manipulation risk
- Wide spreads (3-5%+)

**Revised minimum filters (STRICTER than original GST):**

For VN30 large caps:
âœ“ Average daily value > 50 billion VND (was 2B - INCREASE 25x)
âœ“ Average daily volume > 1,000,000 shares (was 500k - DOUBLE)
âœ“ Traded every single day last 30 days (was 20 - EXTEND)
âœ“ Spread < 0.5% (was 1% - TIGHTEN)

Rationale: With Â±7% limits and auctions, you CANNOT afford to be stuck in illiquid names.
### 8.5 Sector Rotation Patterns in VN

**Common patterns (validate with data):**

```
Banking sector:
- Often leads market up or down
- Affected by interest rate policy
- Q4 and Q1 often strongest (year-end push)

Real estate:
- Cyclical, sensitive to credit policy
- Often lags banking sector moves
- Project completion cycles matter

Steel/Manufacturing:
- Tracks infrastructure and construction
- Raw material prices matter
- Often moves with China economic data

Technology:
- Smaller sector in VN
- More volatile
- Less institutional ownership

Consumer/Retail:
- Defensive characteristics
- Holiday seasons matter (Tet, etc.)
- Steady but slower growth
```

**Sector rotation strategy:**

```
Track sector relative strength:

Sector RS = (Sector Index / VN-Index) Ã— 100

Rising RS = Sector outperforming market
Falling RS = Sector underperforming

Overweight positions in top 2-3 performing sectors.
Avoid bottom 2 performing sectors.

Rebalance monthly.
```

### 8.6 Foreign Ownership Limits

**Room for foreign investors:**

```
Most stocks: 49% foreign ownership limit
Some restrictions: 30% or lower
Some liberalized: 100% allowed

When foreign ownership approaches limit:
- Stock may become "ceiling" for foreign buying
- Can cause technical sell-off
- Creates artificial resistance

Check foreign ownership percentage before entering:
- If > 45%, be cautious
- Sudden sell-off risk if limit approached
- May limit upside

Data usually available on exchange websites or terminal.
```

### 8.7 Tax and Cost Considerations

**Transaction costs:**

```
Brokerage commission: 0.15% - 0.35% (negotiate with broker)
Exchange fee: ~0.01%
Tax: 0.1% on sell transactions

Total round-trip cost: ~0.5% - 0.7%

Implications:
Minimum profit target should exceed transaction costs.
For 0.6% total cost:
Need > 1.5% gross profit to clear costs and make money.

Position sizing:
Avoid tiny positions where costs eat profits.
Minimum position: ~5M VND to make costs reasonable.
```

**Calculation:**

```
Entry: 50,000 Ã— 500 shares = 25,000,000
Buy commission (0.25%): 62,500

Exit: 53,000 Ã— 500 shares = 26,500,000
Sell commission (0.25%): 66,250
Tax (0.1%): 26,500

Total costs: 155,250 VND
Gross profit: 1,500,000
Net profit: 1,344,750
Net return: 5.4% vs 6% gross

Breakeven price: 50,000 + (155,250 / 500) = 50,310
Need > 0.6% move just to break even.
```

---

## IX. MARKET REGIME AWARENESS

### 9.1 Identifying Market Regimes

### 9.1.4 Vietnam-Specific Regime Indicators

**VN-Index historical behavior:**
- 2020 COVID: Crashed to 600s (bear regime validated)
- 2021-2024: Rallied to 1,200+ (bull regime)
- Tends to have 6-12 month trending periods
- Sideways periods are SHORTER than US markets (2-4 months typical)

**Regime detection for VN:**

Current VN-Index = X
200-day MA = Y

If X > Y Ã— 1.10 (10% above MA):
  = STRONG BULL - Aggressive long positions
  = Most TA signals work here
  
If X > Y Ã— 1.00 but < Y Ã— 1.10:
  = MILD BULL - Standard strategy
  
If X < Y Ã— 1.00 but > Y Ã— 0.90:
  = MILD BEAR - Defensive, reduce size 50%
  
If X < Y Ã— 0.90:
  = STRONG BEAR - Minimal trading (cash position)
  = TA signals will whipsaw

**Historical win rates by regime (based on 2009-2012 study):**
- Strong bull (VN-Index > 200-MA by 10%+): RSI/MA strategies ~70% win rate
- Sideways (VN-Index Â±10% of 200-MA): ~45% win rate
- Bear (VN-Index < 200-MA by 10%+): ~30% win rate

**Action:** If VN-Index enters bear regime:
1. Exit 70-80% of positions within 1 week
2. Remaining positions: only highest conviction (score 10+)
3. Move to 80% cash
4. Wait for regime change confirmation
5. Paper trade mean-reversion setups (not momentum)

**Four primary regimes:**

**1. Bull trending (uptrend):**
```
Characteristics:
- VN-Index above 200-day MA
- 50-day MA above 200-day MA (or crossing up)
- Making higher highs and higher lows
- ADX > 25 on weekly chart
- Increasing volume on up days

Strategy:
- Aggressive long positions
- Use standard position sizing (1-2% risk)
- Focus on breakouts and momentum
- Hold winners longer
- Be selective about shorts
```

**2. Bear trending (downtrend):**
```
Characteristics:
- VN-Index below 200-day MA
- 50-day MA below 200-day MA (or crossing down)
- Making lower highs and lower lows
- ADX > 25 on weekly chart
- Increasing volume on down days

Strategy:
- Defensive: small positions, shorter holds
- Reduce risk to 0.5-1% per trade
- Focus on highest quality setups only (score 9+)
- Take profits quickly (1.5R-2R)
- Avoid breakout trades
- Consider cash/sidelines
```

**3. Range-bound (consolidation):**
```
Characteristics:
- VN-Index between defined support/resistance
- ADX < 20
- Choppy price action
- 50-day MA and 200-day MA converging or flat

Strategy:
- Mean reversion trades
- Buy support, sell resistance
- Tight stops
- Quick profit taking (1R-1.5R)
- Smaller position sizes
- Fewer trades overall
```

**4. Transitional (regime change):**
```
Characteristics:
- Major moving average crosses occurring
- Breaking long-term support or resistance
- Volume spikes
- Volatility increases (ATR expanding)

Strategy:
- Reduce position sizes by 50%
- Wait for confirmation of new regime
- Avoid counter-trend trades
- Be patient
- Paper trade to learn new regime characteristics
```

### 9.2 Regime-Based Position Sizing

```
Bull Market:
- Standard scoring: 7 = 1%, 9 = 1.5%, 11 = 2%
- Can have 6-8 concurrent positions
- Aggressive targets: 3R-4R+

Bear Market:
- Conservative scoring: 8 = 0.5%, 10 = 1%, 11 = 1.5%
- Maximum 3-4 concurrent positions
- Conservative targets: 1.5R-2R

Range Market:
- Moderate scoring: 8 = 0.75%, 10 = 1.25%
- Maximum 4-5 concurrent positions
- Quick targets: 1R-2R

Transitional:
- Minimal: Any score = 0.5%
- Maximum 2 positions
- Very quick exits: 1R
```

### 9.3 Adapting Indicators to Regime

**Indicator effectiveness by regime:**

**Bull market:**
- Moving averages: Very effective
- RSI: Less useful (stays overbought)
- MACD: Good for trend confirmation
- Bollinger Bands: Use for pullback entries
- Volume: Critical for confirmation

**Bear market:**
- Moving averages: Good for resistance
- RSI: Somewhat useful for oversold bounces
- MACD: Good for spotting trend exhaustion
- Bollinger Bands: Lower band provides bounce opportunities
- Volume: Watch for climax selling

**Range market:**
- Moving averages: Less effective (choppy crosses)
- RSI: Very useful (true overbought/oversold)
- MACD: Many false signals (ignore)
- Bollinger Bands: Most effective (buy low, sell high)
- Volume: Less important

**Tactical adjustments:**

```
Bull market:
- Loosen stops (2.5-3Ã— ATR)
- Use 20 EMA pullbacks aggressively
- Ignore overbought RSI readings
- Trail winners with 20 EMA

Bear market:
- Tighten stops (1.5-2Ã— ATR)
- Take profits at resistance
- Don't fight the trend
- Exit fast on failed setups

Range market:
- Tightest stops (1-1.5Ã— ATR)
- Fade extremes (RSI <30 buy, >70 sell)
- Quick profit taking
- Avoid trend-following setups
```

### 9.4 VN-Index as Leading Indicator

**Create market filter:**

```
Daily calculation:

VN-Index position relative to 50-day MA:
Score 1: Below 50 MA and falling
Score 2: Below 50 MA but rising  
Score 3: Above 50 MA but falling
Score 4: Above 50 MA and rising

VN-Index ADX:
Score 1: ADX < 20 (no trend)
Score 2: ADX 20-30 (weak trend)
Score 3: ADX 30-40 (moderate trend)
Score 4: ADX > 40 (strong trend)

VN-Index relative to 200-day MA:
Score 1: > 10% below
Score 2: 0-10% below
Score 3: 0-10% above
Score 4: > 10% above

Market Score = Sum of three scores (3-12 possible)

Trading rules based on Market Score:

Score 3-5: Bear market mode - minimal trading
Score 6-7: Defensive - small positions, high selectivity
Score 8-10: Normal - standard strategy
Score 11-12: Aggressive - larger positions, hold longer

Recalculate daily, adjust strategy weekly.
```

---

## X. ADVANCED CONCEPTS

### 10.1 Correlation Management

**Calculating correlation:**

```
For any two stocks, collect last 60 daily returns:

Return = (Today's Close / Yesterday's Close) - 1

Stock A returns: [r1, r2, r3, ..., r60]
Stock B returns: [s1, s2, s3, ..., s60]

Correlation coefficient (Ï):
Ï = Î£[(ri - mean_r) Ã— (si - mean_s)] / âˆš[Î£(ri - mean_r)Â² Ã— Î£(si - mean_s)Â²]

Or use built-in correlation functions in spreadsheet/programming tools.

Result: -1 to +1
+1 = Perfect positive correlation (move together)
0 = No correlation
-1 = Perfect negative correlation (move opposite)
```

**Practical application:**

```
Before adding new position:

1. Calculate correlation of new stock with each existing position
2. Identify highest correlation
3. Apply adjustment:

If max(Ï) < 0.5: No adjustment
If max(Ï) = 0.5-0.7: Reduce new position by 20%
If max(Ï) = 0.7-0.85: Reduce new position by 50%
If max(Ï) > 0.85: Don't add position

4. Calculate portfolio average correlation:

Avg_Ï = Average of all pairwise correlations

If Avg_Ï > 0.6: Portfolio too concentrated
Reduce all positions by 20% or close most correlated position
```

**Sector correlation monitoring:**

```
Banking stocks typically have Ï > 0.7 with each other.
Don't hold more than 2-3 banking stocks simultaneously.

Same for real estate, steel, etc.

Diversification requires:
- Multiple sectors
- Low correlation between positions
```

### 10.2 Expectancy Calculation

**Trade expectancy formula:**

```
Expectancy (E) = (Win Rate Ã— Avg Win) - (Loss Rate Ã— Avg Loss)

Or in R-multiples:
E = (Win Rate Ã— Avg R-win) - (Loss Rate Ã— Avg R-loss)

Where R = initial risk amount

Example calculation from 50 trades:

Wins: 28 trades
Losses: 22 trades
Win rate: 28/50 = 56%
Loss rate: 22/50 = 44%

Winning trades: +2.5R, +1.8R, +3.2R, +1.5R, ... (28 trades)
Avg win: 2.3R

Losing trades: -1R, -0.8R, -1R, -0.7R, ... (22 trades)
Avg loss: -0.9R

E = (0.56 Ã— 2.3R) - (0.44 Ã— 0.9R)
E = 1.288R - 0.396R  
E = 0.892R per trade

Interpretation:
For every trade, you expect to make 0.892R on average.
This is excellent (> 0.3R is good).

Over 100 trades risking 1% each:
Expected profit = 100 Ã— 0.892% = 89.2% total return
```

**Target expectancy:**

```
Minimum viable: E > 0.2R
Good system: E > 0.3R
Excellent system: E > 0.5R
Exceptional: E > 1.0R

If E < 0.2R:
- System needs improvement
- Transaction costs eating profits
- Poor trade selection or execution
```

**Improving expectancy:**

```
To increase expectancy, improve either:

1. Win rate (harder):
   - Better entry timing
   - Higher quality setups only
   - Better market regime filtering

2. Avg win / Avg loss ratio (easier):
   - Let winners run longer (trail stops)
   - Cut losses faster (tighter stops initially)
   - Scale out methodology
   - Better profit target placement

Example:
Current: 50% win rate, 2R avg win, 1R avg loss
E = (0.5 Ã— 2) - (0.5 Ã— 1) = 0.5R

Improve by letting winners run:
New: 50% win rate, 2.8R avg win, 1R avg loss
E = (0.5 Ã— 2.8) - (0.5 Ã— 1) = 0.9R

Improvement: 0.9R vs 0.5R = 80% increase in expectancy
Without changing win rate!
```

### 10.3 Maximum Adverse Excursion (MAE) Analysis

**Concept:**
Track how far against you each trade goes before hitting your target or stop.

**Methodology:**

```
For each trade, record:
- Entry price
- Stop price
- Exit price
- Maximum adverse price during trade (worst price reached)

MAE = Entry price - Maximum adverse price (for longs)

As percentage:
MAE% = (Entry - Max Adverse) / Entry Ã— 100

As R-multiple:
MAE_R = (Entry - Max Adverse) / (Entry - Stop)

Example:
Entry: 50,000
Stop: 47,000 (Risk = 3,000, 6%)
Worst price during trade: 48,500
Exit: 56,000 (Winner)

MAE = 50,000 - 48,500 = 1,500 (3%)
MAE_R = 1,500 / 3,000 = 0.5R

Trade went -0.5R against before becoming +2R winner.
```

**Analysis:**

```
After 50+ trades, create scatter plot:
X-axis: Final R-multiple of trade
Y-axis: MAE_R

Observations:

1. If winners typically have MAE < 0.5R:
   Your stops are positioned well
   Tight stops are not hurting

2. If winners typically have MAE > 1.5R:
   Your stops are too tight
   Consider widening stops to 2-2.5Ã— current distance

3. If losers have MAE > 1.2R before stopping out:
   You're holding losers too long
   Tighten stops or exit faster

4. Clusters:
   - Winning trades cluster at MAE 0.3-0.7R = healthy
   - Losing trades cluster near -1R = good discipline
   - If losing trades scattered from -0.5R to -2R = inconsistent stop management
```

**Practical application:**

```
If most winners have MAE between 0.4R and 0.8R:
Set initial stop at 1.5R distance
This gives enough room for normal pullback
But still protects against real breakdowns

If most winners have MAE < 0.3R:
Can use tighter stops (1R distance)
Trade setup is strong, minimal adverse movement

If many winners have MAE > 1R:
Need wider stops (2-2.5R distance)
Or accept smaller position sizes with wider stops
```

### 10.4 Position Pyramiding (Advanced)

**Concept:**
Add to winning positions as they move in your favor.

**Rules:**

```
Only pyramid when:
1. Original position up at least +1R
2. Trend strongly confirmed (ADX > 30)
3. Weekly trend aligned
4. Volume supporting move
5. No signs of exhaustion

Never pyramid:
- Losing positions (don't average down)
- During weak trends (ADX < 25)
- Against higher timeframe trend
- Into resistance
- Based on hope
```

**Pyramiding structures:**

**Structure 1: Equal sizing**
```
Entry 1: 100 shares at 50,000 (base)
Stop: 47,000

Price reaches 53,000 (+1R), add:
Entry 2: 100 shares at 53,000 (add)
Move stop on both to 50,000 (breakeven on base)

Price reaches 56,000 (+2R from base), add:
Entry 3: 100 shares at 56,000 (add)
Move stop on all to 53,000 (+1R on base)

Total: 300 shares
Avg price: 53,000
Risk on entire position: Protected (stop at 53,000)
```

**Structure 2: Decreasing sizing (pyramid shape)**
```
Entry 1: 100 shares at 50,000
Entry 2: 75 shares at 53,000 (25% smaller)
Entry 3: 50 shares at 56,000 (50% smaller than base)

Total: 225 shares
Avg price: 52,333
Lower average = more profit
Less capital at risk in later adds
```

**Structure 3: Reflection method**
```
Entry 1: 50% position at 50,000
Entry 2: Add 25% at 53,000
Entry 3: Add final 25% at 56,000

Always adding smaller amounts than initial
Total committed over time, not all at once
```

**Stop management when pyramiding:**

```
Critical rule:
Never let a winning pyramid turn into a loser on total position.

After each add:
1. Move stop on ALL shares to at least:
   - Breakeven on original shares, OR
   - Guaranteed small profit on total position

2. Trail stop as position moves further in favor

3. If stop is hit, ALL shares exit together
   (Don't try to manage partial exits on stopped positions)
```

**Calculation example:**

```
Original: 200 shares at 50,000 = 10,000,000 VND
Stop: 47,000
Risk: 3,000 per share = 600,000 total

Price reaches 53,000 (+1R):
Add: 150 shares at 53,000 = 7,950,000 VND
Total: 350 shares, 17,950,000 VND invested
Avg price: 51,286

New stop for all shares: 50,500
This guarantees:
- Profit on original 200 shares: 200 Ã— 500 = 100,000
- Small loss on added 150 shares: 150 Ã— 2,500 = 375,000
- Net: -275,000 (still acceptable as was going to risk 600,000 originally)

Or more conservatively, new stop at 51,500:
- Profit on original: 200 Ã— 1,500 = 300,000
- Loss on added: 150 Ã— 1,500 = 225,000
- Net: +75,000 guaranteed profit on entire position
```

**Risk management:**

```
Total risk across all pyramid additions:
Should not exceed 2Ã— original risk

Original risk: 1% = 1,000,000 VND
Maximum total risk with pyramiding: 2% = 2,000,000 VND

This accounts for:
- Potential slippage
- Gap risk on larger position
- Correlation of multiple entries

Never pyramid more than 2 times (3 entries total max)
More than that becomes overconcentrated
```

---

## XI. SYSTEM VALIDATION & BACKTESTING

### 11.1 Data Requirements

Validation of Core Indicators

MA crossovers, RSI, MACD historically outperformed buy-and-hold on HOSE
One study showed RSI strategies returned ~174% vs ~3% for buy-and-hold (2009-2012)
These tools work best in trending markets, which aligns with GST's ADX filtering

**Minimum dataset:**

```
Duration: 2-3 years of daily data
Should include:
- Bull market period
- Bear market period
- Range-bound period
- At least one crisis/crash event

Data points needed per day:
- Open
- High
- Low
- Close
- Volume
- (Unadjusted AND adjusted for splits/dividends)

Number of stocks:
- VN30: All 30 stocks minimum
- Mid-caps: Additional 20-30 stocks
- Total: 50-60 stocks minimum for statistical significance
```

**Data quality checks:**

```
Remove or correct:
- Days with zero volume (market closed or data error)
- Prices that gap >20% without news (data error)
- Negative prices or volumes
- Dates that don't exist

Verify:
- Splits and dividends adjusted correctly
- Ceiling/floor prices recorded correctly
- Volume is in shares (not lots or contracts)
```

### 11.2 Backtesting Methodology

**Walk-forward analysis (NOT curve-fitting):**

```
Total data: 3 years (2022-2024)

In-sample period (training): 2 years (2022-2023)
Out-of-sample period (testing): 1 year (2024)

Process:
1. Develop system parameters using 2022-2023 data
2. Optimize on 2022-2023 data
3. Lock parameters
4. Test on 2024 data WITHOUT any adjustments
5. Compare in-sample vs out-of-sample performance

Good system:
Out-of-sample performance â‰¥ 70% of in-sample performance

If out-of-sample << in-sample:
System is overfit (curve-fitted)
Will not work in live trading
```

**Rolling window analysis:**

```
Year 1 train â†’ Test on Year 2
Year 2 train â†’ Test on Year 3  
Year 1-2 train â†’ Test on Year 3

Calculate metrics for each test period.
System should be profitable across all test windows.

If profitable only in specific periods:
System is regime-dependent
Need adaptation rules or multiple systems
```

**Monte Carlo simulation:**

```
After backtesting, run Monte Carlo:

1. Take your trade sequence (W, L, W, W, L, L, W, ...)
2. Randomize order 10,000 times
3. Calculate equity curve for each randomization
4. Analyze distribution of results:
   - Worst case: 5th percentile equity curve
   - Best case: 95th percentile equity curve
   - Median case: 50th percentile equity curve

This shows:
- Possible range of outcomes
- Worst likely drawdown
- Probability of ruin

Plan for 10th percentile case (conservative).
```

### 11.3 Key Metrics to Track

**Performance metrics:**

```
1. Expectancy (already covered):
   E = (Win Rate Ã— Avg Win) - (Loss Rate Ã— Avg Loss)
   Target: > 0.3R

2. Profit Factor:
   PF = Gross Profit / Gross Loss
   Target: > 1.5
   
   Example:
   Gross profit: 15,000,000
   Gross loss: 8,000,000
   PF = 15,000,000 / 8,000,000 = 1.875 (Good)

3. Win Rate:
   Win% = Winning Trades / Total Trades Ã— 100
   Target: 45-60%
   
   Higher is not always better.
   60% win rate with 1R avg win = worse than
   40% win rate with 3R avg win

4. Average Win / Average Loss Ratio:
   Ratio = Avg Win / Avg Loss
   Target: > 2.0
   
   Example:
   Avg win: 6.5%
   Avg loss: 2.8%
   Ratio = 6.5 / 2.8 = 2.32 (Good)

5. Maximum Drawdown (DD):
   DD = (Peak Equity - Trough Equity) / Peak Equity Ã— 100
   Target: < 20%
   
   Example:
   Peak: 120,000,000
   Trough: 102,000,000
   DD = (120M - 102M) / 120M = 15% (Acceptable)

6. Sharpe Ratio (risk-adjusted returns):
   SR = (Return - Risk-free rate) / Standard Deviation of Returns
   Target: > 1.0 (good), > 2.0 (excellent)
   
   Calculation:
   Annual return: 25%
   Risk-free rate: 5% (VN T-bill rate)
   Std dev of monthly returns: 8%
   SR = (25% - 5%) / (8% Ã— âˆš12) = 20% / 27.7% = 0.72
   
   (This system needs improvement in risk-adjusted terms)

7. Recovery Factor:
   RF = Net Profit / Max Drawdown
   Target: > 3.0
   
   Example:
   Net profit: 30%
   Max DD: 12%
   RF = 30 / 12 = 2.5 (Acceptable, not great)

8. Payoff Ratio:
   PR = Avg Win / Avg Loss
   Target: > 2.0

9. R-multiple distribution:
   Plot histogram of all trade R-multiples
   Should see:
   - Losing trades cluster near -1R
   - Winning trades spread from +1R to +5R+
   - Long right tail (big winners)

10. Trade duration:
    Avg days in trade
    Should align with your timeframe (weeks = 7-30 days typical)
```

**Efficiency metrics:**

```
11. Time in market:
    % of days capital is deployed
    
    Too low (<50%): Missing opportunities
    Too high (>90%): Overtrading, need selectivity

12. Number of trades:
    Per month/year
    
    For swing trading:
    - 2-4 trades per month = About right
    - >8 trades per month = Likely overtrading
    - <1 trade per month = Too selective or system broken

13. Average holding period:
    Days from entry to exit
    
    Should cluster around 10-25 days for swing trading
    If much shorter: More like day trading
    If much longer: More like position trading

14. Consecutive losses:
    Maximum string of losses in backtest
    
    If max = 7 losses in row:
    Your system needs to handle 10 losses in row live
    (Murphy's Law: live trading is always worse)
```

**Risk metrics:**

```
15. Maximum R-loss:
    Worst single trade in R terms
    
    Should be close to -1R (your stop was respected)
    If worst loss is -2R or -3R: Stops not working
    
16. Risk of Ruin:
    Probability of losing X% of capital
    
    Calculate using:
    - Win rate
    - Payoff ratio
    - % risked per trade
    
    Online calculators available
    Target: <1% risk of 30% drawdown

17. Ulcer Index:
    Measures depth and duration of drawdowns
    
    Lower is better
    Compares systems: prefer lower UI for same return

18. MAE/MFE analysis:
    Maximum Adverse Excursion: How far trades go against you
    Maximum Favorable Excursion: How far trades go for you
    
    Use to optimize stop placement and profit targets
```

### 11.4 Parameter Optimization

### 11.4.1 Vietnam Market Studies (For Reference)

Academic research on Vietnam found:
- MA crossovers: Profitable in trending periods (2009-2012, 2021-2024)
- RSI strategies: Highest returns but also highest volatility
- MACD: Also beat buy-and-hold significantly
- KEY FINDING: All indicators worked in trends, none worked in ranges

**Implication for your backtesting:**
- Your optimized parameters SHOULD show:
  - High returns in 2021-2024 (bull period)
  - Flat/negative returns in 2019-2020 (range/bear)
  - This is EXPECTED, not a failure
  
- If your backtest shows profits in ALL periods equally:
  - You probably overfit the data
  - Redo with walk-forward analysis
  
**Realistic expectations:**
Based on historical studies, expect:
- Bull market: +30% to +50% annual returns (above buy-and-hold)
- Range market: âˆ’5% to +5% returns (match market)
- Bear market: âˆ’10% to âˆ’20% (better than buy-and-hold but still negative)

Your job: Crush it in bulls, survive bears, stay flat in ranges.

**Parameters to test:**

```
Moving averages:
- Fast period: 10, 15, 20, 25, 30
- Slow period: 40, 50, 60, 70
- Test all combinations

RSI:
- Period: 9, 11, 14, 17, 20
- Entry threshold: 35-45 (longs), 55-65 (shorts)

ATR multiplier for stops:
- 1.5, 2.0, 2.5, 3.0

Position sizing:
- Risk per trade: 0.5%, 1%, 1.5%, 2%

Holding period:
- Time stop: 10 days, 15 days, 20 days, 30 days

Profit targets:
- T1: 1.5R, 2R, 2.5R
- T2: 2.5R, 3R, 3.5R
```

**Optimization process:**

```
1. Choose one parameter to optimize
2. Hold all others constant
3. Test range of values
4. Record metrics for each value
5. Choose value with best risk-adjusted returns (Sharpe, not just total return)
6. Lock that parameter
7. Move to next parameter
8. Repeat

CRITICAL: 
Don't choose parameters that work best in backtest.
Choose parameters that:
- Work consistently across different periods
- Make logical sense
- Are robust (small changes don't collapse performance)

Example:
Testing MA fast period:

Period 15: Sharpe 1.2, DD 18%, Trades 45
Period 20: Sharpe 1.4, DD 15%, Trades 38 â† Choose this
Period 25: Sharpe 1.5, DD 22%, Trades 31

Even though 25 has higher Sharpe, it has unacceptable DD.
20 is more balanced.
```

**Avoiding over-optimization:**

```
Red flags for curve-fitting:

1. Parameter is very specific:
   "Works best at exactly 19.3 days"
   vs
   "Works well between 18-22 days"
   
   Second is robust, first is curve-fit.

2. Minor parameter change crashes performance:
   Period 20: 30% return
   Period 21: -5% return
   
   This is not robust, don't trust it.

3. Too many parameters:
   >5-6 optimized parameters = likely overfit

4. In-sample much better than out-of-sample:
   In-sample: 40% return
   Out-of-sample: 8% return
   
   System won't work live.

5. Works in only one time period:
   2022: -10%
   2023: +85%
   2024: -5%
   
   Not consistent, probably luck.
```

### 11.5 Slippage and Transaction Costs

**Realistic modeling:**

```
For each trade in backtest, add:

1. Entry slippage:
   Signal triggers at close: 50,000
   Actual fill next day: 50,150 (0.3% worse)
   
   Model: Entry = Signal price Ã— 1.003 (for longs)

2. Exit slippage:
   Similar: Exit = Signal price Ã— 0.997

3. Commission:
   Round-trip: 0.5-0.7% depending on broker

4. Market impact (for larger positions):
   If position > 1% of daily volume:
   Add 0.5-1% additional slippage

Total drag: 1-2% per round trip

This significantly impacts results.
System with 25% return before costs
Might be only 18-20% after costs with 50 trades/year.
```

**Calculation in backtest:**

```
Gross profit on trade: 3,000,000
Entry: 50,000 Ã— 500 shares
Exit: 56,000 Ã— 500 shares

Costs:
Entry slippage: 50,000 Ã— 0.003 Ã— 500 = 75,000
Entry commission: 25,000,000 Ã— 0.0025 = 62,500
Exit slippage: 56,000 Ã— 0.003 Ã— 500 = 84,000
Exit commission: 28,000,000 Ã— 0.0025 = 70,000
Tax: 28,000,000 Ã— 0.001 = 28,000

Total costs: 319,500

Net profit: 3,000,000 - 319,500 = 2,680,500

Gross return: 12%
Net return: 10.7%
Cost drag: 1.3 percentage points
```

### 11.6 Live Trading vs Backtest Expectations

**Adjustment factors:**

```
Backtest shows:
- 30% annual return
- 15% max drawdown
- 55% win rate

Expect in live trading:
- 20-25% annual return (70-80% of backtest)
- 18-22% max drawdown (120-150% of backtest)
- 50-52% win rate (slightly lower)

Why differences:

1. Psychological factors (not in backtest):
   - Fear of losing
   - Greed holding too long
   - Revenge trading
   - Fatigue/distraction

2. Execution issues:
   - Missing fills
   - Platform downtime
   - Fat-finger errors
   - Delayed signals

3. Market adaptation:
   - Market changes after your backtest period
   - Regime shifts
   - New regulations

4. Unknown unknowns:
   - Black swan events
   - Your specific bad luck timing

Rule of thumb:
Plan for 70-80% of backtest performance
Be pleasantly surprised if you match it
```

---

## XII. TRADE EXECUTION & JOURNAL

### 12.1 Pre-Trade Checklist

**Before entering ANY trade, verify:**

```
â˜ Market regime identified (bull/bear/range/transition)
â˜ VN-Index trend aligned with trade direction
â˜ Trade scores â‰¥ 7 on scorecard (â‰¥8 if bear market)
â˜ Liquidity filters passed:
   â˜ Average turnover > minimum
   â˜ No zero-volume days
   â˜ Spread < threshold
â˜ No earnings or major events within 5 days
â˜ Position sizing calculated:
   â˜ Account risk % determined
   â˜ Stop distance measured
   â˜ Number of shares calculated
   â˜ Position value < 25% of capital
â˜ Entry price determined (limit order price set)
â˜ Stop loss price determined and order ready
â˜ Profit targets calculated (T1, T2, T3)
â˜ Risk/reward ratio â‰¥ 2:1 confirmed
â˜ Correlation check done (if other positions open)
â˜ Aggregate portfolio risk < 6%
â˜ Not entering due to FOMO or revenge trading
â˜ Trade thesis written down (why entering)
```

**If any item fails:** Don't take the trade. Wait for better setup.

### 12.2 Order Types and Execution

**Order types in VN market:**

**1. Limit Order (LO):**
```
Specifies exact price you're willing to pay

Buy limit: 50,000
- Will fill at 50,000 or better (lower)
- Might not fill if market doesn't reach your price

Use for:
- Non-urgent entries
- Setting entry price at specific support level
- Avoiding slippage
- Stop loss orders (stop-limit)

Risk: Order might not fill, miss the trade
```

**2. Market Order (MP - Market Price):**
```
Fills at best available price immediately

Use for:
- Urgent exits (stop loss hit)
- Emergency situations
- High liquidity stocks where slippage is minimal

Risk: Might get filled at significantly worse price
Avoid in low liquidity stocks or during auctions
```

**3. ATO/ATC Orders:**
```
ATO (At-the-opening): Participates in opening auction
ATC (At-the-closing): Participates in closing auction

Use sparingly:
- Can get unpredictable fills
- Subject to manipulation
- Wide spreads possible

Generally avoid for systematic trading
```

**Order execution strategy:**

```
For entries:
Use limit orders at your planned entry price
If not filled by end of session:
- Decide if you chase or wait
- Don't chase more than 1% above planned entry
- If setup still valid, can try again next day

For stops:
Use stop-limit orders:
- Stop price: Your stop level (e.g., 47,000)
- Limit price: Slightly worse (e.g., 46,700)
- This triggers limit order when stop price reached
- Limit price ensures you don't get catastrophic fill

For profit targets:
Use limit orders at your target prices:
- T1 at 2R target
- T2 at 3R target
- Adjust as price approaches

For trailing stops:
Manually adjust limit stop order daily:
- Move up to follow 20 EMA or swing lows
- Set limit 0.5-1% below stop price

For emergency exits:
Use market order
Accept slippage as cost of quick exit
```

### 12.3 Trade Journal Template

**Essential information to record for EVERY trade:**

```
TRADE #: [Unique ID]
DATE ENTERED: [DD/MM/YYYY]
TIME: [HH:MM]

STOCK: [Ticker]
ENTRY PRICE: [X,XXX]
POSITION SIZE: [XXX shares]
POSITION VALUE: [XX,XXX,XXX VND]
% OF CAPITAL: [X.X%]

STOP LOSS: [X,XXX]
RISK PER SHARE: [X,XXX]
TOTAL RISK: [X,XXX,XXX VND]
RISK AS % OF CAPITAL: [X.X%]

PROFIT TARGETS:
T1 (2R): [X,XXX] - [XX%] position
T2 (3R): [X,XXX] - [XX%] position
T3 (Trail): 20 EMA / Swing low

EXPECTED REWARD: [X,XXX,XXX VND]
R:R RATIO: [X.X:1]

SETUP TYPE: [Pullback / Breakout / MA Cross / Other]
TIMEFRAME: Daily
PATTERN: [Describe pattern seen]

INDICATORS AT ENTRY:
- 20 EMA: [Above/Below/At]
- 50 EMA: [Above/Below/At]
- RSI(14): [XX]
- MACD: [Positive/Negative/Crossing]
- ADX: [XX]
- Volume: [XX percentile]

MARKET CONTEXT:
VN-Index: [Position relative to MAs, trend]
Market Regime: [Bull/Bear/Range/Transition]
Sector: [Which sector]
Sector Strength: [Strong/Weak/Neutral]

TRADE SCORE: [X/10]
Breakdown:
- Trend: [X/3]
- Setup: [X/3]
- Momentum: [X/2]
- Risk/Reward: [X/2]

CORRELATION WITH EXISTING POSITIONS:
[List other open positions and correlation coefficients]

AGGREGATE PORTFOLIO RISK: [X.X%]
NUMBER OF OPEN POSITIONS: [X]

TRADE THESIS (Why entering):
[Write 2-3 sentences explaining why this is a good trade]
[What you expect to happen]
[Key support/resistance levels]

ENTRY TRIGGER:
[Specific catalyst that triggered entry - candle pattern, volume spike, etc.]

EMOTIONAL STATE: [Calm / Anxious / Excited / Fearful / Confident / Other]
PREPARATION: [Rushed / Planned / Reviewed]

---

[During trade, update this section as events occur:]

TRADE UPDATES:
[Date] - [Event/Action/Observation]
[Date] - [Event/Action/Observation]

---

EXIT INFORMATION:

DATE EXITED: [DD/MM/YYYY]
TIME: [HH:MM]
DAYS HELD: [XX days]

EXIT PRICES:
T1: [X,XXX] at [Date] - [XX shares]
T2: [X,XXX] at [Date] - [XX shares]
Final: [X,XXX] at [Date] - [XX shares]

EXIT REASON:
[Stop hit / Target hit / Time stop / Thesis invalid / Other]

MAXIMUM FAVORABLE EXCURSION (MFE):
Highest price reached: [X,XXX]
MFE: [X,XXX] ([+X.X%] or [+X.XR])

MAXIMUM ADVERSE EXCURSION (MAE):
Lowest price reached: [X,XXX]
MAE: [X,XXX] ([-X.X%] or [-X.XR])

PROFIT/LOSS:
Gross P/L: [X,XXX,XXX VND]
Commissions: [XXX,XXX VND]
Net P/L: [X,XXX,XXX VND]
Return %: [Â±X.X%]
R-multiple: [Â±X.XR]

ACTUAL vs PLANNED:
Planned risk: [X.X%]
Actual risk taken: [X.X%]
Planned hold: [XX days]
Actual hold: [XX days]
Planned exit: [Targets]
Actual exit: [What happened]

POST-TRADE ANALYSIS:

WHAT WENT RIGHT:
[List things that worked as planned]

WHAT WENT WRONG:
[List mistakes, unexpected events, errors]

LESSONS LEARNED:
[Key takeaways from this trade]

EXECUTION QUALITY (1-10): [X]
Entry timing: [X/10]
Exit timing: [X/10]
Risk management: [X/10]
Emotional control: [X/10]

WOULD I TAKE THIS TRADE AGAIN?
[Yes / No / With modifications]

Why?
[Explanation]

TAGS: [#pullback #banking #bull-market #winner #stopped-out]
```

### 12.4 Weekly and Monthly Reviews

**Weekly review (every weekend):**

```
WEEK OF: [Date range]

TRADES THIS WEEK:
Number of trades: [X]
Winners: [X]
Losers: [X]
Break-even: [X]

Win rate: [XX%]

PERFORMANCE:
Starting capital: [XXX,XXX,XXX]
Ending capital: [XXX,XXX,XXX]
Change: [Â±X.X%]

Best trade: [+X.XR] on [Ticker]
Worst trade: [-X.XR] on [Ticker]

Average winner: [+X.XR]
Average loser: [-X.XR]
Expectancy: [Â±X.XR]

POSITIONS:
Currently open: [X positions]
Aggregate risk: [X.X%]
Largest position: [Ticker, X.X% of capital]
Sector exposure: [List]

MARKET CONTEXT:
VN-Index performance: [Â±X.X%]
Market regime: [Bull/Bear/Range]
Any regime changes noted?
Volatility: [High/Normal/Low]

SYSTEM ADHERENCE:
Trades that followed rules: [X/X]
Trades that broke rules: [X/X]
Rule violations: [Describe]

EMOTIONAL ASSESSMENT:
Overall emotional state: [1-10 scale]
Stress level: [Low/Medium/High]
Confidence: [Low/Medium/High]

OBSERVATIONS:
[What patterns did you notice?]
[What worked well?]
[What didn't work?]
[Any market shifts?]

ACTION ITEMS FOR NEXT WEEK:
[ ] [Specific action 1]
[ ] [Specific action 2]
[ ] [Specific action 3]
```

**Monthly review (first weekend of each month):**

```
MONTH OF: [Month Year]

OVERALL PERFORMANCE:
Starting capital: [XXX,XXX,XXX]
Ending capital: [XXX,XXX,XXX]
Return: [Â±X.X%]
Benchmark (VN-Index): [Â±X.X%]
Alpha: [Â±X.X%]

Peak equity: [XXX,XXX,XXX]
Trough equity: [XXX,XXX,XXX]
Maximum drawdown: [X.X%]

TRADE STATISTICS:
Total trades: [XX]
Winners: [XX] ([XX%])
Losers: [XX] ([XX%])

Average winner: [+X.XR] ([+X.X%])
Average loser: [-X.XR] ([-X.X%])
Largest winner: [+X.XR] on [Ticker]
Largest loser: [-X.XR] on [Ticker]

Win/Loss ratio: [X.X]
Expectancy: [Â±X.XR]
Profit factor: [X.X]

SETUP ANALYSIS:
[Create table:]

Setup Type | Trades | Win% | Avg R | Total R
Pullback   |   12   | 58%  | +0.9R | +10.8R
Breakout   |    8   | 50%  | +0.3R | +2.4R
MA Cross   |    5   | 40%  | -0.2R | -1.0R

Best setup: [Pullback]
Worst setup: [MA Cross]
Action: [Reduce MA Cross trades, focus on pullbacks]

HOLDING PERIOD ANALYSIS:
Average hold: [XX days]
Shortest: [XX days]
Longest: [XX days]
Optimal range: [XX-XX days] (based on winners)

TIME DECAY ANALYSIS:
Trades held <10 days: [Win% = XX%]
Trades held 10-20 days: [Win% = XX%]
Trades held 20-30 days: [Win% = XX%]
Trades held >30 days: [Win% = XX%]

ENTRY TIMING:
Morning session entries: [Win% = XX%]
Afternoon session entries: [Win% = XX%]
Best time: [Morning]

EXIT ANALYSIS:
Hit T1: [XX trades]
Hit T2: [XX trades]
Hit T3: [XX trades]
Stopped out: [XX trades]
Time stop: [XX trades]

Should targets be adjusted? [Analysis]

SECTOR PERFORMANCE:
[Create table:]

Sector      | Trades | Win% | Avg R
Banking     |   8    | 63%  | +1.2R
Real Estate |   5    | 40%  | -0.3R
Steel       |   4    | 75%  | +1.8R

Best sector: [Steel]
Worst sector: [Real Estate]
Action: [Increase Steel exposure, avoid Real Estate]

MARKET REGIME PERFORMANCE:
Bull market: [X trades, XX% win, +X.XR avg]
Bear market: [X trades, XX% win, +X.XR avg]
Range: [X trades, XX% win, +X.XR avg]

Best in: [Bull market]
Needs work in: [Bear market]

RISK MANAGEMENT ASSESSMENT:
Largest position taken: [X.X%]
Largest loss: [X.X%]
Maximum aggregate risk: [X.X%]

Times exceeded risk limits: [X]
Average actual risk taken: [X.X%] (Plan: X.X%)

Risk discipline: [Excellent / Good / Needs improvement]

RULE VIOLATIONS:
Total violations: [X]

Types:
- Entered without confirmation: [X times]
- Exceeded position size: [X times]
- Moved stop in wrong direction: [X times]
- Revenge traded: [X times]
- [Other]: [X times]

Cost of violations: [-X.XR total]

Corrective actions needed:
[Specific steps to improve]

PSYCHOLOGY ASSESSMENT:
Best trading state: [Describe]
Worst trading state: [Describe]
Trigger for poor decisions: [Identify]

Number of emotional trades: [X]
Impact: [Cost in R-multiples]

Improvement plan:
[Specific steps]

MAE/MFE ANALYSIS:
Average MAE on winners: [X.XR]
Average MAE on losers: [X.XR]
Average MFE on winners: [X.XR]
Average MFE on losers: [X.XR]

Interpretation:
[Are stops positioned correctly?]
[Are we letting winners run enough?]
[Should we tighten or widen stops?]

SYSTEM EFFECTIVENESS:
Scorecard accuracy:
Score 7-8: [XX% win]
Score 9-10: [XX% win]
Score 11+: [XX% win]

Should scoring be adjusted? [Analysis]

Parameters working well:
- [List]

Parameters needing adjustment:
- [List with proposed changes]

GOALS vs ACTUAL:
Planned trades this month: [XX]
Actual trades: [XX]
Variance: [Â±X%]

Planned return: [X%]
Actual return: [X%]
Variance: [Â±X%]

LESSONS LEARNED:
1. [Key lesson 1]
2. [Key lesson 2]
3. [Key lesson 3]

GOALS FOR NEXT MONTH:
1. [Specific goal 1]
2. [Specific goal 2]
3. [Specific goal 3]

SYSTEM CHANGES TO IMPLEMENT:
[ ] [Change 1]
[ ] [Change 2]
[ ] [Change 3]

PERSONAL DEVELOPMENT:
[ ] [Area to improve]
[ ] [Skill to develop]
[ ] [Knowledge to acquire]

---

QUARTERLY REVIEW (every 3 months):
- Compare system performance vs benchmark
- Analyze if market regime has fundamentally changed
- Consider major system adjustments
- Review overall capital allocation
- Assess if continue with system or pivot

ANNUAL REVIEW (end of year):
- Full year performance summary
- Sharpe ratio, Sortino ratio calculations
- Compare to initial backtest expectations
- Decide on system modifications for next year
- Set annual goals for next year
- Consider increasing/decreasing capital allocation
```

---

## XIII. PSYCHOLOGICAL FRAMEWORK

### 13.1 The Trading Psychology Loop

**Understanding the cycle:**

```
Event â†’ Thought â†’ Emotion â†’ Action â†’ Result â†’ New Event

Example of good cycle:
Trade hits stop (-1R) â†’ "This is just one trade" â†’ Calm â†’ Follow next setup â†’ Win (+2.5R) â†’ Confidence

Example of bad cycle:
Trade hits stop (-1R) â†’ "I can't win" â†’ Frustration â†’ Revenge trade â†’ Bigger loss (-2R) â†’ Despair â†’ More revenge â†’ Cascade of losses
```

**Breaking bad cycles:**

```
After a loss:
1. Close platform for 15-30 minutes
2. Physical reset (walk, breathe, water)
3. Review trade objectively (journal)
4. Identify: Was it system error or random variation?
5. If system error: Learn and adjust
6. If random variation (proper trade): Accept and move on
7. Return when calm

Never:
- Trade immediately after emotional event
- Increase size to "make it back"
- Abandon system after losses
- Skip stops because "this one will work"
```

### 13.2 Common Psychological Traps

**1. Recency bias:**
```
Trap: Recent results overly influence future decisions
Last 3 trades were winners â†’ "I'm on fire, increase size"
Last 3 trades were losers â†’ "System is broken, abandon it"

Reality:
Streaks are normal statistical variation
3 wins doesn't mean you've figured it out
3 losses doesn't mean system is broken

Solution:
Focus on 30-50 trade samples
Keep risk consistent regardless of recent results
Trust the process, not the outcome
```

**2. Loss aversion:**
```
Trap: Pain of loss felt 2-3x stronger than pleasure of equal gain
Holding losers hoping they'll come back
Cutting winners too early to "lock in profit"

Example:
Position at -5%, should stop out â†’ "Let me give it another day"
Position at +5%, should hold for target â†’ "I'll take the profit now"

Solution:
Set stops and targets BEFORE entering
Use automatic orders (remove emotion)
Remember: Big winners come from holding, not selling at first profit
```

**3. Overconfidence after wins:**
```
Trap: Success attributed to skill, failure to bad luck
After big win â†’ "I'm better than I thought, increase size"
After series of wins â†’ "I can trade larger / more frequently"

Reality:
Some wins are luck
Market conditions may have been favorable
Increased confidence = increased risk-taking = eventual blowup

Solution:
Maintain standard position sizing even after wins
Take partial profits off table after exceptional month
Review what was luck vs skill
Stay humble
```

**4. Revenge trading:**
```
Trap: Need to "make it back" immediately after loss
Trade hits stop â†’ Immediately enter another trade
Larger size, lower quality setup
Often results in second loss â†’ even more desperate

Solution:
Mandatory break after stopped out (15-30 minutes minimum)
Cannot increase size after loss (only decrease or maintain)
Cannot enter new trade within 1 hour of loss
Cannot trade same stock that stopped you out same day
```

**5. Analysis paralysis:**
```
Trap: Perfect setup paralysis
Waiting for 100% certainty before entering
Missing trade after trade
When finally entering, it's too late

Reality:
No setup is perfect
7-8/10 score is enough
Waiting for 10/10 means never trading

Solution:
If score â‰¥ 7, take trade
Don't look for additional confirmation
Trust system
Some losers are inevitable and acceptable
```

**6. Anchoring:**
```
Trap: Fixating on irrelevant prices
Stock you considered at 45,000 now at 52,000 â†’ "Too expensive, I'll wait"
Your old entry at 50,000, now stopped out, price back to 50,000 â†’ "Won't buy where I was stopped"

Reality:
Past prices are irrelevant
Only current setup matters
What price "should" be doesn't matter

Solution:
Evaluate each setup fresh
Don't care where price was
Don't care where you entered before
Only: Is setup valid NOW?
```

### 13.3 Emotional State Management

**Pre-trading routine:**

```
Before market opens:

â˜ Adequate sleep (7+ hours)
â˜ Physical exercise (even 10 minutes)
â˜ Healthy meal
â˜ Review trading plan
â˜ Check positions and stops
â˜ Identify potential setups
â˜ Emotional state check: Am I calm and focused?

If not calm:
- Don't trade
- Paper trade only
- Use smaller size
- More conservative (only best setups)

Red flag emotional states (don't trade):
- Anxious about money
- Angry from non-trading event
- Euphoric from recent wins
- Desperate to make money
- Distracted by life issues
```

**During trading hours:**

```
Regular checks every hour:

Physical state:
- Shoulders tense? â†’ Relax
- Jaw clenched? â†’ Release
- Breathing shallow? â†’ Deep breaths
- Posture slumped? â†’ Straighten

Emotional state:
- Feeling rushed? â†’ Slow down
- Feeling invincible? â†’ Review risk limits
- Feeling scared? â†’ Reduce size or close positions
- Feeling bored? â†’ Don't create trades, be patient

If emotional:
- Stand up, walk 5 minutes
- Do NOT make trading decisions
- Come back when neutral
```

**Post-trading routine:**

```
After market closes:

â˜ Update journal (all trades)
â˜ Review executions (any errors?)
â˜ Plan for tomorrow
â˜ Disconnect from markets
â˜ Engage in non-trading activities

Evening:
- No screen time 1 hour before bed
- No market checking
- No planning "perfect" next trade
- Quality sleep for next day

If bad day:
- Extra thorough journal entry
- Identify lesson
- Talk to mentor/trading buddy
- Early sleep
- Fresh start tomorrow
```

### 13.4 Building Mental Resilience

**Handling drawdowns:**

```
In drawdown (down from peak):

Week 1 (-3% from peak):
- Acknowledge: This is normal
- Review: Any system errors?
- Adjust: If needed, reduce size 20%
- Continue: Execute plan

Week 2 (-5% from peak):
- Acknowledge: Still within normal parameters
- Review: Market regime shift?
- Adjust: Reduce size 30%
- Continue: Only best setups (score â‰¥8)

Week 3 (-8% from peak):
- Acknowledge: Serious drawdown
- Review: Detailed trade analysis
- Adjust: Reduce size 50%
- Consider: Take week break

Week 4 (-10%+ from peak):
- STOP trading
- Full system review
- Paper trade only
- Psychological reset
- Don't return until clarity restored
```

**Positive self-talk scripts:**

```
After loss:
"This is one trade out of hundreds. My system has positive expectancy. This loss is already accounted for in my plan. I executed correctly. Next trade."

During losing streak:
"Losing streaks are statistical reality. My system is designed to handle this. I will continue following rules. The edge will reassert itself. Patience."

After big win:
"This is great, but just one trade. Don't change anything. Don't get overconfident. Don't increase size. Stay process-focused."

During temptation to break rules:
"I know breaking rules feels good in the moment but leads to long-term failure. My rules exist for a reason. I will follow them. Trust the process."

When market not cooperating:
"I cannot control the market. I can only control my process. Some periods have fewer trades. That's okay. Quality over quantity. Patience is part of the edge."
```

**Visualization exercises:**

```
Daily (5 minutes):

1. Close eyes
2. Visualize perfect trade execution:
   - Setup appears
   - Evaluate calmly
   - Enter with confidence
   - Manage position emotionally neutral
   - Exit at target, celebrate briefly, move on
3. Visualize handling loss perfectly:
   - Stop is hit
   - Feel brief disappointment
   - Quickly move to analysis
   - Identify no errors
   - Close journal
   - Ready for next opportunity

Repeat mental rehearsal of ideal behavior.
Brain doesn't distinguish well between visualization and reality.
This builds positive neural pathways.
```

---

## XIV. IMPLEMENTATION ROADMAP

### 14.1 Phase 1: Foundation (Weeks 1-2)

**Objectives:**
- Understand all concepts
- Set up tools
- Collect data
- No live trading yet

**Tasks:**

```
Week 1:
â˜ Read entire system documentation (this document)
â˜ Set up brokerage account (if not done)
â˜ Set up charting platform
â˜ Configure indicators:
   - Moving averages (20 EMA, 50 EMA on daily)
   - 200 SMA on weekly
   - RSI (14)
   - MACD (12,26,9)
   - ATR (14)
   - Volume
â˜ Create spreadsheet for journal
â˜ Collect 2-3 years historical data for VN30 stocks

Week 2:
â˜ Complete data collection (OHLCV for 50+ stocks)
â˜ Set up backtesting environment (code or tool)
â˜ Create scorecard template (for trade evaluation)
â˜ Write out personal trading rules in own words
â˜ Identify 5 stocks to focus on initially
â˜ Study historical charts (practice pattern recognition)
â˜ No trading yet
```

### 14.2 Phase 2: Backtesting (Weeks 3-6)

**Objectives:**
- Validate system on historical data
- Optimize parameters for VN market
- Build confidence in system
- Still no live trading

**Tasks:**

```
Week 3:
â˜ Backtest basic system on 5 stocks:
   - Use default parameters (20/50 EMA, RSI 14, etc.)
   - Track all metrics
   - Record every trade in journal template
â˜ Calculate:
   - Expectancy
   - Win rate
   - Profit factor
   - Maximum drawdown
   - Sharpe ratio

Week 4:
â˜ Expand backtest to 20 stocks
â˜ Test parameter variations:
   - Moving average periods
   - RSI periods
   - ATR multipliers
â˜ Compare results
â˜ Select optimal parameters for VN market

Week 5:
â˜ Full backtest on 50+ stocks with chosen parameters
â˜ Test across different market regimes:
   - Bull period
   - Bear period
   - Range period
â˜ Calculate all performance metrics
â˜ Create Monte Carlo simulations

Week 6:
â˜ Out-of-sample testing:
   - Use different time period than optimization
   - Validate system still works
â˜ Final parameter selection
â˜ Document system rules clearly
â˜ Review: Is expectancy >0.3R?
â˜ Review: Is max drawdown <20%?
â˜ If YES â†’ Proceed to Phase 3
â˜ If NO â†’ Revise system, repeat Phase 2
```

### 14.3 Phase 3: Paper Trading (Weeks 7-10)

**Objectives:**
- Practice execution without risk
- Build habit and routine
- Identify practical issues
- Still no real money

**Tasks:**

```
Week 7:
â˜ Set up paper trading account (track on paper or simulator)
â˜ Start with 100,000,000 VND virtual capital
â˜ Follow complete routine:
   - Pre-market preparation
   - During-market monitoring
   - After-market journaling
â˜ Take every valid setup (score â‰¥7)
â˜ Use real prices (including slippage estimates)
â˜ Execute as if real money

Week 8-10:
â˜ Continue paper trading
â˜ Aim for 20-30 paper trades total
â˜ Full journal for each trade
â˜ Weekly and monthly reviews
â˜ Track:
   - Actual vs expected results
   - Execution errors
   - Emotional reactions
   - Time requirements

After 20+ paper trades, evaluate:
â˜ Expectancy close to backtest? (>70% of backtest)
â˜ Able to follow rules consistently?
â˜ Comfortable with process?
â˜ Understand time commitment?

If YES to all â†’ Proceed to Phase 4
If NO to any â†’ Continue paper trading until comfortable
```

### 14.4 Phase 4: Live Trading - Probation (Weeks 11-16)

**Objectives:**
- Real money, but small size
- Prove system with actual risk
- Learn from live market
- Build confidence

**Tasks:**

```
Week 11:
â˜ Fund account with 30% of intended capital
   (If plan is 100M, start with 30M)
â˜ Risk only 0.5% per trade (more conservative)
â˜ Maximum 2-3 concurrent positions
â˜ Take first live trade
â˜ Detailed journal entry

Week 12-16:
â˜ Continue live trading
â˜ Aim for 15-25 live trades during this phase
â˜ Strict adherence to rules
â˜ Weekly reviews (every weekend)
â˜ Track:
   - Actual slippage vs estimates
   - Fill rates
   - Emotional responses to real losses
   - Any system modifications needed

After 15+ live trades:
â˜ Calculate actual expectancy
â˜ Compare to backtest and paper trading
â˜ Assess psychological comfort
â˜ Review all rule violations

If performing well (expectancy >0.2R, following rules):
â˜ Proceed to Phase 5

If struggling:
â˜ Back to paper trading
â˜ Identify issues
â˜ Resolve before continuing
```

### 14.5 Phase 5: Full Implementation (Week 17+)

**Objectives:**
- Trade with full capital
- Standard position sizing
- Ongoing optimization
- Long-term execution

**Tasks:**

```
Week 17:
â˜ Increase capital to 60% of intended
â˜ Risk 1% per trade (standard)
â˜ Up to 4-5 concurrent positions
â˜ Continue rigorous journaling

Month 5-6:
â˜ Increase to 100% intended capital
â˜ Risk 1-2% per trade based on score
â˜ Up to 6-8 concurrent positions
â˜ Monthly reviews
â˜ Start quarterly reviews

Ongoing (every month):
â˜ Review all trades
â˜ Update metrics
â˜ Adjust parameters if needed (based on data)
â˜ Refine system based on learnings
â˜ Continue education
â˜ Track market regime changes

Quarterly:
â˜ Comprehensive system review
â˜ Compare to benchmarks
â˜ Assess if major changes needed
â˜ Update trading plan document
â˜ Consider capital adjustments

Annually:
â˜ Full year review
â˜ Sharpe ratio calculation
â˜ Compare to initial goals
â˜ Major system modifications if needed
â˜ Set goals for next year
â˜ Celebrate successes
â˜ Learn from failures
```

---

## XV. APPENDICES

### 15.1 Quick Reference - Trade Entry Criteria

**Minimum requirements for ANY trade:**

```
âœ“ Market regime identified and strategy adjusted
âœ“ Trade scores â‰¥ 7 on scorecard (â‰¥8 in bear market)
âœ“ Liquidity filters passed
âœ“ No major events within 5 days
âœ“ Stop loss determined (â‰¤7% or â‰¤2Ã—ATR)
âœ“ Risk/reward â‰¥ 2:1
âœ“ Position sized correctly (1-2% risk)
âœ“ Aggregate portfolio risk < 6%
âœ“ Written trade thesis
âœ“ Not acting on emotion
```

### 15.2 Quick Reference - Exit Criteria

**Exit immediately if:**

```
âœ— Stop loss hit
âœ— Profit target hit
âœ— Trade thesis invalidated
âœ— Major negative news
âœ— Technical breakdown (support broken)
âœ— Daily loss limit hit
âœ— Emotional compromise
```

### 15.3 Quick Reference - Risk Limits

```
Per trade: 1-2% maximum
Per day: Stop if -2% (stop new trades), exit all if -3%
Per week: Reduce sizes if -5%, stop if -7%
Per month: Paper trade if -10%, stop if -15%
Maximum drawdown: Stop all trading if -20% from peak
Aggregate portfolio: â‰¤6% total risk across all positions
Single position: â‰¤25% of capital
Sector exposure: â‰¤40% of capital
Open positions: 6-8 maximum
Consecutive losses: Reduce size after 3, stop after 7
```

### 15.4 Common Mistakes Checklist

**Avoid these errors:**

```
â˜ Trading without stop loss
â˜ Moving stop away from entry (wider)
â˜ Risking >2% on single trade
â˜ Position size >25% of capital
â˜ Trading illiquid stocks
â˜ Chasing price (entering after big move)
â˜ Revenge trading after loss
â˜ Increasing size after loss
â˜ Trading on emotion
â˜ Entering without confirmation
â˜ Exiting winners too early
â˜ Holding losers too long
â˜ Trading during news events
â˜ Over-leveraging with correlated positions
â˜ Not journaling trades
â˜ Skipping reviews
â˜ Abandoning system during drawdown
â˜ Overtrading (too many positions)
â˜ Trading while distracted
â˜ Not following scoring system
```

### 15.5 Emergency Protocols

**If experiencing extreme stress or major losses:**

```
IMMEDIATE ACTIONS:
1. Close all positions (market orders)
2. Step away from computer
3. Do NOT trade for 48 hours minimum
4. Physical activity (exercise)
5. Call mentor or trusted person
6. Sleep on it

AFTER 48 HOURS:
1. Review what went wrong (journal)
2. Identify specific errors
3. Create correction plan
4. Paper trade for 2 weeks minimum
5. See if errors corrected
6. Return with 50% capital and 0.5% risk
7. Rebuild confidence slowly

NEVER:
- Trade larger to "make it back"
- Skip your trading rules
- Abandon proven system
- Give up after losses
```

---

## XVI. FINAL THOUGHTS

### 16.1 The Reality of Trading

```
Trading is:
âœ“ 80% psychology and discipline
âœ“ 20% system and analysis

Success requires:
âœ“ Patience (good setups are rare)
âœ“ Discipline (follow rules even when uncomfortable)
âœ“ Humility (market will humble you)
âœ“ Consistency (execute plan every time)
âœ“ Resilience (handle losses and drawdowns)
âœ“ Continuous learning (market evolves)

Trading is NOT:
âœ— Get rich quick
âœ— Easy money
âœ— Exciting all the time
âœ— High win rate guarantee
âœ— Perfect system exists
```

### 16.2 Key Principles to Remember

```
1. Process over outcome
   - Focus on executing well, not on single trade P/L
   - One trade proves nothing
   - 50-100 trades show truth

2. Position sizing is everything
   - Proper size = survive drawdowns
   - Too large = eventual ruin
   - When in doubt, smaller

3. Let winners run, cut losers short
   - Hardest thing to do
   - Goes against human nature
   - Essential for profitability

4. Market regime matters
   - Don't fight the market
   - Adapt strategy to conditions
   - Cash is a position

5. Trading is a marathon
   - Compound returns over years
   - Consistency beats home runs
   - Survive first, profit second

6. Journal everything
   - Can't improve what you don't measure
   - Patterns emerge from data
   - Your best teacher

7. System > Ego
   - Follow rules even when "feeling" is different
   - Trust process, not intuition
   - Discipline beats intelligence
```

### 16.3 Continuous Improvement

```
Never stop:
- Reading (books, articles, studies)
- Analyzing (your trades and others')
- Testing (new ideas on paper first)
- Adapting (market evolves constantly)
- Learning (from both wins and losses)

But also never:
- Change system mid-drawdown
- Abandon rules because "this time is different"
- Overcomplicate what works
- Fix what isn't broken
```