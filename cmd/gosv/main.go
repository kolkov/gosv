package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kolkov/gosv/internal/config"
	"github.com/kolkov/gosv/internal/supervisor"
)

func ensureConfigExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		defaultConfig := `processes:
  - name: "example-process"
    command: "cmd.exe"
    args: ["/c", "echo Hello World && timeout /t 30 /nobreak"]
    autostart: true
    autorestart: "always"
    stop_signal: "SIGKILL"
    stop_wait: 5s

  - name: "ping-test"
    command: "ping"
    args: ["localhost", "-n", "30"]
    autostart: true
    autorestart: "always"
`
		if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
		log.Printf("[INFO] Created default config at %s", path)
	}
	return nil
}

func main() {
	cfgPath := flag.String("c", "gsv.yaml", "Path to configuration file")
	tuiMode := flag.Bool("tui", false, "Enable terminal UI mode")
	debugMode := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Проверяем и создаем конфиг при необходимости
	if err := ensureConfigExists(*cfgPath); err != nil {
		log.Fatalf("[ERROR] %v", err)
	}

	// Загрузка конфигурации
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("[ERROR] Config load failed: %v", err)
	}

	// Инициализация супервизора
	sv := supervisor.New(cfg)

	// Устанавливаем логгер для отладки
	if *debugMode {
		sv.SetLogger(func(log string) {
			fmt.Println(log)
		})
	}

	// Запуск всех процессов с autostart
	if err := sv.StartAll(); err != nil {
		log.Fatalf("[ERROR] Startup failed: %v", err)
	}
	log.Println("[INFO] Supervisor started")

	// Краткая задержка для запуска процессов
	time.Sleep(500 * time.Millisecond)

	// Выводим начальный статус
	sv.PrintStatus()

	// Если включен TUI режим
	if *tuiMode {
		// Запускаем TUI интерфейс
		sv.RunTUI()
	} else {
		// Режим без TUI
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

		log.Println("Entering signal handling loop. Press Ctrl+C to exit.")

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case sig := <-sigCh:
				switch sig {
				case syscall.SIGHUP:
					log.Println("[INFO] Reloading config...")
					if newCfg, err := config.Load(*cfgPath); err == nil {
						sv.ReloadConfig(newCfg)
						log.Println("[INFO] Config reloaded successfully")
						sv.PrintStatus()
					} else {
						log.Printf("[ERROR] Config reload failed: %v", err)
					}
				default:
					log.Printf("[INFO] Received %s, shutting down...", sig)
					sv.StopAll()
					log.Println("[INFO] Supervisor stopped")
					return
				}
			case <-ticker.C:
				log.Println("Periodic status update")
				sv.PrintStatus()
			}
		}
	}

	// Этот код достижим только если вышли из TUI
	sv.StopAll()
	log.Println("[INFO] Supervisor stopped")
}
