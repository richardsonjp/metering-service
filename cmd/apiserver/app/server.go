package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"metering-service/cmd/apiserver/app/routes"
	"metering-service/cmd/apiserver/app/store"
	"metering-service/config"
	"metering-service/pkg/utils/logs"
)

func Run() {
	cfg := config.Load()

	loc, err := time.LoadLocation(cfg.System.TimeZone)
	if err != nil {
		panic("Invalid timezone: " + cfg.System.TimeZone)
	}
	time.Local = loc

	logs.Init(cfg.Log.Level)

	appStore := store.Init(cfg)
	defer appStore.Shutdown()

	server := routes.NewHTTPServer(appStore)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logs.Log.Infof("[Server:Addr]: %s", cfg.System.Addr)

		if err := server.Listen(cfg.System.Addr); err != nil {
			logs.Log.Errorf("Server stopped: %v", err)
			quit <- syscall.SIGTERM
		}
	}()

	<-quit
	logs.Log.Warn("Shutdown signal received...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.ShutdownWithContext(ctx); err != nil {
		logs.Log.Errorf("Graceful shutdown error: %v", err)
	}

	logs.Log.Warn("Server gracefully stopped.")
}
