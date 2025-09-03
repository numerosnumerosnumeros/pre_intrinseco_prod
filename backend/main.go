package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/patrickmn/go-cache"
	"github.com/stripe/stripe-go/v81"
	"go.uber.org/zap"

	"nodofinance/middleware"
	"nodofinance/routes/app"
	"nodofinance/routes/auth"
	"nodofinance/routes/payments"
	"nodofinance/utils/env"
	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"
	"nodofinance/utils/static"
)

func main() {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Warning: Could not determine executable path: %v", err)
	}

	execDir := filepath.Dir(execPath)
	err = godotenv.Load(filepath.Join(execDir, ".env"))
	if err != nil {
		log.Fatalf("Warning: Could not load .env file: %v", err)
	}

	devMode := false
	if os.Getenv("DEV_MODE") == "true" {
		devMode = true
	}

	log.Printf("Starting server in dev mode: %t", devMode)

	logger.InitLogger(devMode, execDir)
	defer logger.Log.Sync()
	logger.Log.Info("Logger initialized")

	dataCache := cache.New(5*time.Minute, 10*time.Minute) // default expiration, cleanup interval
	// jwt validation uses its own cache
	logger.Log.Info("Cache initialized")

	ctx := context.Background()
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Log.Fatal("Unable to load SDK config")
	}

	ssm := ssm.NewFromConfig(sdkConfig)
	c := cognitoidentityprovider.NewFromConfig(sdkConfig)
	s := s3.NewFromConfig(sdkConfig)
	d := dynamodb.NewFromConfig(sdkConfig)
	logger.Log.Info("AWS SDK initialized")

	/*
		nodofinance_table:
			- PK: username
			- SK: composite_sk:
				* TICKER#{ticker} -> attributes: last_update, currency, analysis
				* FINANCE#{ticker}#{reverse_year}#{period_order} -> attributes: financial data fields
				* METADATA -> attributes: stripe_id, expires_date, ctokens
	*/

	err = env.LoadEnv(ssm, "/")
	if err != nil {
		logger.Log.Fatal("Failed to load environment variables", zap.Error(err))
	}
	logger.Log.Info("Environment variables loaded successfully")

	if err := env.ValidateAllVars(); err != nil {
		logger.Log.Fatal("Environment validation failed", zap.Error(err))
	}

	err = jwt.LoadJWKS()
	if err != nil {
		logger.Log.Fatal("Failed to load JWKS", zap.Error(err))
	}
	logger.Log.Info("JWKS loaded successfully")

	err = static.CacheFrontend(execDir)
	if err != nil {
		logger.Log.Fatal("Failed to cache frontend files", zap.Error(err))
	}
	logger.Log.Info("Frontend files cached successfully")

	stripeKey, ok := env.Get("STRIPE_SK")
	if !ok {
		logger.Log.Fatal("STRIPE_SK not found in environment variables")
	}
	stripe.Key = stripeKey
	logger.Log.Info("Stripe initialized successfully")

	openAIkey, ok := env.Get("OPENAI_API_KEY")
	if !ok {
		logger.Log.Fatal("OPENAI_API_KEY not found in environment variables")
	}
	ai := openai.NewClient(
		option.WithAPIKey(openAIkey),
	)

	mux := http.NewServeMux()

	// *
	// **
	// ***
	// ****
	// ***** HEALTH
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// *
	// **
	// ***
	// ****
	// ***** AUTH
	mux.HandleFunc("/api/auth/token", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			auth.Token(w, r, c, d)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: false,
			CheckPremium:    false,
		},
	))

	mux.HandleFunc("/api/auth/signup", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			auth.SignUp(w, r, c)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: false,
			CheckPremium:    false,
		},
	))

	mux.HandleFunc("/api/auth/signin", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			auth.SignIn(w, r, c, d)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: false,
			CheckPremium:    false,
		},
	))

	mux.HandleFunc("/api/auth/signout", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			auth.SignOut(w, r)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: false,
			CheckPremium:    false,
		},
	))

	mux.HandleFunc("/api/auth/verify", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			auth.Verify(w, r, c)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: false,
			CheckPremium:    false,
		},
	))

	mux.HandleFunc("/api/auth/resend", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			auth.Resend(w, r, c)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: false,
			CheckPremium:    false,
		},
	))

	mux.HandleFunc("/api/auth/reset", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			auth.Reset(w, r, c)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: false,
			CheckPremium:    false,
		},
	))

	mux.HandleFunc("/api/auth/forgot", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			auth.Forgot(w, r, c)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: false,
			CheckPremium:    false,
		},
	))

	// *
	// **
	// ***
	// ****
	// ***** PAYMENTS
	mux.HandleFunc("/api/auth/payments/create-checkout-session", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			payments.Checkout(w, r, d, dataCache)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: true,
			CheckPremium:    false,
		},
	))

	mux.HandleFunc("/api/auth/payments/webhooks", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			payments.Webhooks(w, r, c, d)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: false,
			CheckPremium:    false,
		},
	))

	mux.HandleFunc("/api/auth/payments/create-portal-session", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			payments.Portal(w, r, d)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: true,
			CheckPremium:    true,
		},
	))

	mux.HandleFunc("/api/auth/payments/refresh-status", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			payments.Refresh(w, r, c, d, dataCache)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: true,
			CheckPremium:    false,
		},
	))

	// *
	// **
	// ***
	// ****
	// ***** APP
	mux.HandleFunc("/api/app/mount", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			app.MountPortfolio(w, r, d, dataCache)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"GET"},
			DevMode:         devMode,
			ValidateRequest: true,
			CheckPremium:    true,
		},
	))

	mux.HandleFunc("/api/app/url", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			app.URL(w, r)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: true,
			CheckPremium:    true,
		},
	))

	mux.HandleFunc("/api/app/submit", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			app.Submit(w, r, d, s, ai, dataCache, devMode)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: true,
			CheckPremium:    true,
		},
	))

	mux.HandleFunc("/api/app/mount-ticker", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			app.MountTicker(w, r, d)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"GET"},
			DevMode:         devMode,
			ValidateRequest: true,
			CheckPremium:    true,
		},
	))

	mux.HandleFunc("/api/app/analyst", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			app.Analyst(w, r, d, ai)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"POST"},
			DevMode:         devMode,
			ValidateRequest: true,
			CheckPremium:    true,
		},
	))

	mux.HandleFunc("/api/app/edit", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			app.Edit(w, r, d)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"PATCH"},
			DevMode:         devMode,
			ValidateRequest: true,
			CheckPremium:    true,
		},
	))

	mux.HandleFunc("/api/app/delete-ticker", middleware.Filter(
		func(w http.ResponseWriter, r *http.Request) {
			app.Delete(w, r, d, dataCache)
		},
		middleware.FilterConfig{
			AllowedMethods:  []string{"DELETE"},
			DevMode:         devMode,
			ValidateRequest: true,
			CheckPremium:    true,
		},
	))

	// *
	// **
	// ***
	// ****
	// ***** STATIC
	mux.HandleFunc("/", static.Handler(devMode, []string{"GET", "HEAD"}))

	port := 80

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           http.MaxBytesHandler(mux, 100*1024), // 100KB limit for request body
		ReadHeaderTimeout: 3 * time.Second,                     // Timeout for reading headers
		ReadTimeout:       20 * time.Second,                    // Timeout for reading the request body
		WriteTimeout:      60 * time.Second,                    // Timeout for writing the response
		IdleTimeout:       60 * time.Second,                    // Timeout for idle connections (keep-alive)
		MaxHeaderBytes:    32 * 1024,                           // 32KB limit for headers
	}

	logger.Log.Info("Server starting on", zap.Int("port", port))
	logger.Log.Fatal("Server failed", zap.Error(server.ListenAndServe()))
}
