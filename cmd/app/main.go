package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/api"
	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/db"
	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/handler"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/notification"
	"github.com/nonobeam/golang-stock-trading/internal/redis"
	"github.com/nonobeam/golang-stock-trading/internal/regime/ftd"
	"github.com/nonobeam/golang-stock-trading/internal/risk"
	"github.com/nonobeam/golang-stock-trading/internal/router"
	"github.com/nonobeam/golang-stock-trading/internal/service"
	"github.com/nonobeam/golang-stock-trading/internal/service/account"
	"github.com/nonobeam/golang-stock-trading/internal/service/auth"
	"github.com/nonobeam/golang-stock-trading/internal/service/jwt"
	"github.com/nonobeam/golang-stock-trading/internal/service/market"
	"github.com/nonobeam/golang-stock-trading/internal/service/monitor"
	"github.com/nonobeam/golang-stock-trading/internal/service/otp"
	"github.com/nonobeam/golang-stock-trading/internal/service/position"
	"github.com/nonobeam/golang-stock-trading/internal/service/recommendation"
	"github.com/nonobeam/golang-stock-trading/internal/service/scanner"
	signalservice "github.com/nonobeam/golang-stock-trading/internal/service/signal"
	"github.com/nonobeam/golang-stock-trading/internal/service/telegram"
	"github.com/nonobeam/golang-stock-trading/internal/service/watchlist"
	"github.com/nonobeam/golang-stock-trading/internal/signals"
	"github.com/nonobeam/golang-stock-trading/internal/websocket"
	"github.com/nonobeam/golang-stock-trading/proto/ml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}

	logger.Init(cfg)
	logger.Info().Msg("Starting golang-stock-trading application")
	logger.Info().
		Str("appName", cfg.AppName).
		Str("env", cfg.AppEnv).
		Str("apiBaseUrl", cfg.DnseApiBaseUrl).
		Msg("Configuration loaded")

	// === Initialize Database ===
	dbConn, err := initDatabase(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer dbConn.Close()

	// === Initialize Repositories ===
	positionRepo := repository.NewPositionRepository(dbConn)
	userConfigRepo := repository.NewUserConfigRepository(dbConn)
	watchlistRepo := repository.NewWatchlistRepository(dbConn)
	signalRepo := repository.NewSignalHistoryRepository(dbConn)
	stockPrefRepo := repository.NewStockSignalPrefRepository(dbConn)
	regimeRepo := ftd.NewRepository(dbConn)

	// === Initialize ML Service Client ===
	var mlClient ml.MLPredictionServiceClient
	mlHost := os.Getenv("ML_SERVICE_HOST")
	mlPort := os.Getenv("ML_SERVICE_PORT")
	if mlHost != "" && mlPort != "" {
		mlAddress := fmt.Sprintf("%s:%s", mlHost, mlPort)
		mlConn, err := grpc.Dial(mlAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Error().Err(err).Msg("Failed to connect to ML service")
		} else {
			logger.Info().Str("address", mlAddress).Msg("Connected to ML service")
			defer mlConn.Close()
			mlClient = ml.NewMLPredictionServiceClient(mlConn)
		}
	} else {
		logger.Warn().Msg("ML_SERVICE_HOST or ML_SERVICE_PORT not set, ML features disabled")
	}

	// === Initialize Services ===
	marketService := market.NewService(nil) // Websocket client updated later
	defer marketService.Stop()

	accountService := account.NewService(positionRepo, userConfigRepo)
	positionService := position.NewService(positionRepo)
	signalService := signalservice.NewService(signalRepo)
	watchlistService := watchlist.NewService(watchlistRepo)
	recommendationService := recommendation.NewService(signalRepo, positionRepo, marketService, mlClient)

	// === Initialize Telegram Bot ===
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var telegramBot *telegram.BotService
	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != 0 {
		telegramBot, err = telegram.NewBotService(cfg, userConfigRepo)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to initialize Telegram bot")
		} else {
			// Configure bot with adapters
			adapters := NewBotAdapters(accountService, positionService)
			telegramBot.SetRiskManager(adapters)
			telegramBot.SetPositionTracker(adapters)

			// Wire repositories and services for status command
			telegramBot.SetPositionRepository(positionRepo)
			telegramBot.SetWatchlistRepository(watchlistRepo)
			telegramBot.SetRegimeRepository(regimeRepo) // Wire FTD status
			if mlClient != nil {
				telegramBot.SetMLClient(mlClient)
			}

			telegramBot.Start(ctx)
			telegramBot.Broadcast("<b>Stock Trading Bot Started</b>\n\nI won't notify you until trading token is needed.")
			logger.Info().Msg("Telegram bot started")
		}
	} else {
		logger.Warn().Msg("Telegram not configured, trading token will need manual input")
	}

	// === Initialize Redis ===
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to Redis - Redis is mandatory for caching")
	}
	defer redisClient.Close()

	// === Initialize OTP Service ===
	otpService := otp.NewService(telegramBot)

	// === Initialize DNSE Auth Service ===
	dnseAuth := api.NewDNSEAuthService(cfg.DnseApiBaseUrl, cfg.DnseUsername, cfg.DnsePassword)

	// === Initialize JWT Service ===
	jwtService := jwt.NewService(dnseAuth)

	// === Initialize API Server ===
	// Start API server after all dependencies are initialized
	go func() {
		startAPIServer(cfg, marketService, accountService, positionService, signalService, watchlistService, recommendationService, stockPrefRepo, otpService, jwtService)
	}()

	authService := auth.NewAuthService(cfg, redisClient)
	emailService := notification.NewEmailService(cfg)
	_ = emailService

	// Default simplified user ID
	defaultUserID := int64(1)

	// === Register RestartHandler with Telegram Bot ===
	if telegramBot != nil {
		restartHandler := RestartHandlerFunc(func(ctx context.Context, otp string) error {
			// Exchange OTP for trading token
			_, err := authService.GetTradingToken(ctx, defaultUserID, otp)
			return err
		})
		telegramBot.SetRestartHandler(restartHandler)
		logger.Info().Msg("Restart handler registered with Telegram bot")
	}

	if cfg.DnseUsername != "" && cfg.DnsePassword != "" {
		logger.Info().Msg("=== Starting Authentication Flow ===")

		loginResp, err := authService.Login(cfg.DnseUsername, cfg.DnsePassword)
		if err != nil {
			logger.Error().Err(err).Msg("Login failed")
		} else {
			logger.Info().
				Strs("roles", loginResp.Roles).
				Msg("Login successful")

			if telegramBot != nil {
				maxAttempts := 3
				var tradingTokenReceived bool

				// Check if we have a valid trading token in Redis first
				cachedToken, err := redisClient.GetTradingToken(ctx, defaultUserID)
				if err == nil && cachedToken != "" {
					logger.Info().Msg("Found cached trading token in Redis")
					authService.SetTradingToken(cachedToken)
					tradingTokenReceived = true

					// Initialize APIs
					infoAPI := api.NewInfoAPI(authService.GetClient())
					tradingAPI := api.NewTradingAPI(authService.GetClient(), authService.GetTradingTokenValue())

					logger.Info().Msg("Info and Trading APIs initialized with cached token")
					_ = infoAPI
					_ = tradingAPI
				}

				for attempt := 1; attempt <= maxAttempts && !tradingTokenReceived; attempt++ {
					attemptsLeft := maxAttempts - attempt

					// Request OTP from Telegram
					otpVal, err := otpService.Request(ctx, 2*time.Minute)
					if err != nil {
						logger.Error().Err(err).Int("attempt", attempt).Msg("Failed to get OTP")
						if attemptsLeft > 0 {
							telegramBot.Broadcast(fmt.Sprintf("Failed to get OTP: %s\n\n%d attempts remaining.", err.Error(), attemptsLeft))
						} else {
							telegramBot.Broadcast("Failed to get OTP after all attempts.")
						}
						break
					}

					// Exchange OTP for trading token via API
					tradingTokenResp, err := authService.GetTradingToken(ctx, defaultUserID, otpVal)
					if err != nil {
						logger.Error().Err(err).Int("attempt", attempt).Msg("Failed to exchange OTP for trading token")
						
						if attemptsLeft > 0 {
							telegramBot.Broadcast(fmt.Sprintf("Failed to get trading token: %s\n\nPlease send a NEW OTP. %d attempts remaining.", err.Error(), attemptsLeft))
						} else {
							telegramBot.Broadcast("Failed to get trading token after all attempts. Please restart the application.")
						}
						continue // Try again with new OTP
					}

					// Success!
				tradingTokenReceived = true
				logger.Info().Int("expiresIn", tradingTokenResp.ExpiresIn).Msg("Trading token received from API")
				
				// Calculate expiration time
				expirationTime := time.Now().Add(time.Duration(tradingTokenResp.ExpiresIn) * time.Second)
				expirationStr := expirationTime.Format("2006-01-02 15:04:05")
				durationStr := formatDuration(tradingTokenResp.ExpiresIn)
				
				telegramBot.Broadcast(fmt.Sprintf("✅ <b>Trading token received successfully!</b>\n\n⏱ <b>Expires in:</b> %s\n📅 <b>Expiration time:</b> %s", durationStr, expirationStr))

					// Initialize APIs
					infoAPI := api.NewInfoAPI(authService.GetClient())
					tradingAPI := api.NewTradingAPI(authService.GetClient(), authService.GetTradingTokenValue())

					logger.Info().Msg("Info and Trading APIs initialized")
					_ = infoAPI
					_ = tradingAPI
				}
			} else {
				logger.Warn().Msg("Telegram bot not available, cannot request Smart OTP")
			}
		}
	} else {
		logger.Warn().Msg("DNSE credentials not configured, skipping login demo")
	}

	logger.Info().Msg("=== Setting up WebSocket ===")
	wsClient := websocket.NewClient(cfg)
	wsClient.SetToken(authService.GetAccessToken())

	// === Initialize Market Data Service ===
	marketDataService := service.NewMarketDataService(wsClient)
	if err := marketDataService.Start(); err != nil {
		logger.Error().Err(err).Msg("Failed to start market data service")
	}
	defer marketDataService.Stop()

	// === Initialize Price Monitor Service ===
	if telegramBot != nil {
		// Wire market data service for /status command
		telegramBot.SetMarketDataService(marketDataService)
		
		monitorService := monitor.NewPriceMonitorService(marketDataService, telegramBot, watchlistRepo)
		monitorService.Start(ctx)
		logger.Info().Msg("Price Monitor Service started and running...")
	} else {
		logger.Warn().Msg("Price Monitor Service skipped because Telegram Bot is waiting for restart")
		// NOTE: In a real scenario, we might want to start it anyway and inject bot later, 
		// but since NewPriceMonitorService requires *BotService, we defer it.
		// Actually botService is nil if not configured. If configured but waiting for token, it is not nil.
		// The check 'if telegramBot != nil' handles the case where bot token is missing in config.
	}

	// Update market service with WS client
	// marketService.SetWebSocketClient(wsClient) // If such method exists

	wsClient.RegisterStockInfoHandler(func(data *websocket.StockInfo) {
		logger.Info().
			Str("symbol", data.Symbol).
			Float64("price", data.MatchPrice).
			Float64("change", data.ChangeRate).
			Msg("Stock update received")
	})

	wsClient.RegisterMarketIndexHandler(func(data *websocket.MarketIndex) {
		logger.Info().
			Str("index", data.IndexName).
			Float64("value", data.IndexValue).
			Msg("Index update received")
	})

	// === Initialize Live Scanner ===
	var liveScanner *scanner.LiveScanner
	if telegramBot != nil {
		logger.Info().Msg("=== Initializing Live Scanner ===")
		
		// 1. Create Signal Scanner (with default strategies)
		signalScanner := signals.NewDefaultSignalScanner()

		// 2. Create Position Sizer
		// Note: We use a default capital for alerts, actual trading uses user specific capital
		positionSizer := risk.NewPositionSizer(100_000_000) // 100M default for estimates

		// 3. Create Live Scanner
		scannerCfg := &scanner.Config{
			DB:               dbConn,
			WSClient:         wsClient,
			SignalScanner:    signalScanner,
			PositionSizer:    positionSizer,
			BotService:       telegramBot,
			MinScore:         cfg.ScannerMinScore,
			MinScoreForAlert: cfg.ScannerMinAlertScore,
			MinBars:          cfg.ScannerMinBars,
			BarCacheSize:     300, // Keep 300 bars history
			RegimeRepo:       regimeRepo,
		}

		liveScanner = scanner.NewLiveScanner(scannerCfg)
		
		// Start scanner in background (it will load watchlist and subscribe)
		go func() {
			// Wait a bit for WS capability to be ready
			time.Sleep(2 * time.Second)
			if err := liveScanner.Start(); err != nil {
				logger.Error().Err(err).Msg("Failed to start Live Scanner")
			}
		}()
	} else {
		logger.Warn().Msg("Live Scanner skipped because Telegram Bot is waiting for restart")
	}

	logger.Info().Msg("Application running. Press Ctrl+C to exit.")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down...")
	cancel()
	
	if liveScanner != nil {
		liveScanner.Stop()
	}
	
	wsClient.Close()
	if telegramBot != nil {
		telegramBot.Broadcast("Bot shutting down...")
	}
	logger.Info().Msg("Application stopped")
}

func startAPIServer(
	cfg *config.Config,
	marketService *market.Service,
	accountService *account.Service,
	positionService *position.Service,
	signalService *signalservice.Service,
	watchlistService *watchlist.Service,
	recommendationService *recommendation.Service,
	stockPrefRepo *repository.StockSignalPrefRepository,
	otpService *otp.Service,
	jwtService *jwt.Service,
) {
	// Initialize handlers
	handlerDeps := router.HandlerDeps{
		MarketHandler:         handler.NewMarketHandler(marketService),
		AccountHandler:        handler.NewAccountHandler(accountService),
		PositionHandler:       handler.NewPositionHandler(positionService),
		SignalHandler:         handler.NewSignalHandler(signalService),
		WatchlistHandler:      handler.NewWatchlistHandler(watchlistService),
		RecommendationHandler: handler.NewRecommendationHandler(recommendationService),
		StockPrefHandler:      handler.NewStockPrefHandler(stockPrefRepo),
		OTPHandler:            handler.NewOTPHandler(otpService),
		JWTHandler:            handler.NewJWTHandler(jwtService),
	}

	// Setup router
	r := router.NewRouter(handlerDeps, cfg)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info().Str("port", cfg.ServerPort).Msg("API server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed to start")
		}
	}()
}

// initDatabase initializes the database connection
func initDatabase(cfg *config.Config) (*sql.DB, error) {
	dbConfig := &db.Config{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            5432,
		User:            getEnv("DB_USER", "trading_user"),
		Password:        getEnv("DB_PASSWORD", "trading_pass"),
		DBName:          getEnv("DB_NAME", "trading"),
		Schema:          getEnv("DB_SCHEMA", "public"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxConnections:  25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}
	
	database, err := db.NewDatabase(dbConfig)
	if err != nil {
		return nil, err
	}
	return database.DB, nil
}


// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// formatDuration converts seconds to human-readable duration string
func formatDuration(seconds int) string {
	duration := time.Duration(seconds) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	
	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%d hours %d minutes", hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d hours", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	return fmt.Sprintf("%d seconds", seconds)
}
