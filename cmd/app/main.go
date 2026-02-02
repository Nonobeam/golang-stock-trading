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
	signalservice "github.com/nonobeam/golang-stock-trading/internal/service/signal"
	"github.com/nonobeam/golang-stock-trading/internal/service/telegram"
	"github.com/nonobeam/golang-stock-trading/internal/service/watchlist"
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
		telegramBot, err = telegram.NewBotService(cfg)
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
			if mlClient != nil {
				telegramBot.SetMLClient(mlClient)
			}

			telegramBot.Start(ctx)
			telegramBot.SendMessage("<b>Stock Trading Bot Started</b>\n\nI won't notify you until trading token is needed.")
			logger.Info().Msg("Telegram bot started")
		}
	} else {
		logger.Warn().Msg("Telegram not configured, trading token will need manual input")
	}

	// === Initialize Redis ===
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to Redis")
		// Continue without Redis - will fall back to Telegram only
	}
	if redisClient != nil {
		defer redisClient.Close()
	}



	// === Initialize OTP Service ===
	otpService := otp.NewService(redisClient, telegramBot)

	// === Initialize DNSE Auth Service ===
	dnseAuth := api.NewDNSEAuthService(cfg.DnseApiBaseUrl, cfg.DnseUsername, cfg.DnsePassword)

	// === Initialize JWT Service ===
	jwtService := jwt.NewService(redisClient, dnseAuth)

	// === Initialize API Server ===
	// Start API server after all dependencies are initialized
	go func() {
		startAPIServer(cfg, marketService, accountService, positionService, signalService, watchlistService, recommendationService, stockPrefRepo, otpService, jwtService)
	}()

	authService := auth.NewAuthService(cfg)
	emailService := notification.NewEmailService(cfg)
	_ = emailService

	// === Register RestartHandler with Telegram Bot ===
	if telegramBot != nil {
		restartHandler := RestartHandlerFunc(func(ctx context.Context, otp string) error {
			// Delete old OTP from cache
			otpService.Delete(ctx)

			// Store new OTP in cache
			otpService.Set(ctx, otp)

			// Exchange OTP for trading token
			_, err := authService.GetTradingToken(otp)
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

				for attempt := 1; attempt <= maxAttempts && !tradingTokenReceived; attempt++ {
					attemptsLeft := maxAttempts - attempt

					// Try to get OTP from Redis first, or request from Telegram
					otpVal, err := otpService.GetOrRequest(ctx, 2*time.Minute)
					if err != nil {
						logger.Error().Err(err).Int("attempt", attempt).Msg("Failed to get OTP")
						if attemptsLeft > 0 {
							telegramBot.SendMessage(fmt.Sprintf("Failed to get OTP: %s\n\n%d attempts remaining.", err.Error(), attemptsLeft))
						} else {
							telegramBot.SendMessage("Failed to get OTP after all attempts.")
						}
						break
					}

					// Exchange OTP for trading token via API
					tradingTokenResp, err := authService.GetTradingToken(otpVal)
					if err != nil {
						logger.Error().Err(err).Int("attempt", attempt).Msg("Failed to exchange OTP for trading token")
						// OTP might be invalid - delete from Redis
						otpService.Delete(context.Background())

						if attemptsLeft > 0 {
							telegramBot.SendMessage(fmt.Sprintf("Failed to get trading token: %s\n\nPlease send a NEW OTP. %d attempts remaining.", err.Error(), attemptsLeft))
						} else {
							telegramBot.SendMessage("Failed to get trading token after all attempts. Please restart the application.")
						}
						continue // Try again with new OTP
					}

					// Success!
					tradingTokenReceived = true
					logger.Info().Int("expiresIn", tradingTokenResp.ExpiresIn).Msg("Trading token received from API")
					telegramBot.SendMessage("Trading token received successfully!")

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

	logger.Info().Msg("Application running. Press Ctrl+C to exit.")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down...")
	cancel()
	wsClient.Close()
	if telegramBot != nil {
		telegramBot.SendMessage("Bot shutting down...")
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
