// Command Semantic serves vector similarity search over a MongoDB Atlas
// vector index, with embeddings produced by the sidecar Python service.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aekam27/Semantic/pkg/auth_service"
	"github.com/aekam27/Semantic/pkg/cache_service"
	"github.com/aekam27/Semantic/pkg/config"
	"github.com/aekam27/Semantic/pkg/connections"
	"github.com/aekam27/Semantic/pkg/dao"
	"github.com/aekam27/Semantic/pkg/middleware"
	"github.com/aekam27/Semantic/pkg/router"
	"github.com/aekam27/Semantic/pkg/semantic_search_service"
	"github.com/aekam27/Semantic/pkg/token"
)

// shutdownGrace is how long in-flight requests get to finish on SIGTERM.
const shutdownGrace = 15 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dialCtx, cancelDial := context.WithTimeout(ctx, 30*time.Second)
	defer cancelDial()

	mongo, err := connections.NewMongo(dialCtx, cfg.Mongo)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := mongo.Close(closeCtx); err != nil {
			log.Printf("mongo disconnect: %v", err)
		}
	}()

	redisCache := connections.NewRedisCache(cfg.Redis)
	tokenStore := cache_service.NewTokenStore(redisCache, time.Hour)

	issuer, err := token.NewIssuer(cfg.JWTSecret, cfg.TokenTTL)
	if err != nil {
		return err
	}

	authenticator := middleware.NewAuthenticator(issuer, tokenStore)
	authenticator.StartRefresh(ctx)

	httpClient := &http.Client{Timeout: cfg.RequestTimeout}
	handler := router.NewRouter(
		authenticator,
		auth_service.NewHandler(issuer, tokenStore, authenticator.Refresh),
		semantic_search_service.NewHandler(
			dao.InitializeMongoDAO(mongo, cfg.Mongo.Collection),
			cfg.EmbeddingServiceURL,
			httpClient,
		),
	)

	// Explicit timeouts: the previous http.ListenAndServe used none, leaving
	// the server open to connections that never finish sending a request.
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Println("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
