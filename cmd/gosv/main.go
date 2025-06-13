package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kolkov/gosv/internal/config"
	"github.com/kolkov/gosv/internal/supervisor"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfgPath := "gosv.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("[ERROR] Config load failed: %v", err)
	}

	sv := supervisor.New(cfg)
	if err := sv.StartAll(); err != nil {
		log.Fatalf("[ERROR] Startup failed: %v", err)
	}
	log.Println("[INFO] Supervisor started")
	sv.PrintStatus()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				sv.PrintStatus()
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				log.Println("[INFO] Reloading config...")
				if newCfg, err := config.Load(cfgPath); err == nil {
					sv.ReloadConfig(newCfg)
					log.Println("[INFO] Config reloaded successfully")
					sv.PrintStatus()
				} else {
					log.Printf("[ERROR] Config reload failed: %v", err)
				}
			default:
				log.Printf("[INFO] Received %s, shutting down...", sig)
				sv.StopAll()
				cancel()

				// Даем время на graceful shutdown
				time.Sleep(1 * time.Second)
				sv.PrintStatus()
				log.Println("[INFO] Supervisor stopped")
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
