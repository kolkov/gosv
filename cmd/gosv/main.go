package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kolkov/gosv/internal/config"
	"github.com/kolkov/gosv/internal/process"
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
	// Глобальные флаги
	cfgPath := flag.String("c", "gsv.yaml", "Path to configuration file")
	tuiMode := flag.Bool("tui", false, "Enable terminal UI mode")
	debugMode := flag.Bool("debug", false, "Enable debug logging")

	// Флаги управления процессами
	startProc := flag.String("start", "", "Start specific process")
	stopProc := flag.String("stop", "", "Stop specific process")
	restartProc := flag.String("restart", "", "Restart specific process")
	runProc := flag.String("run", "", "Run process in foreground mode")
	listProcs := flag.Bool("list", false, "List all configured processes")
	status := flag.Bool("status", false, "Show current status")
	reload := flag.Bool("reload", false, "Reload configuration")

	flag.Parse()

	// Проверка и создание конфига при необходимости
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

	// Обработка команд управления процессами
	switch {
	case *listProcs:
		listAllProcesses(cfg)
		return

	case *status:
		sv.PrintStatus()
		return

	case *reload:
		handleReload(sv, cfgPath)
		return

	case *startProc != "":
		if err := sv.StartProcess(*startProc); err != nil {
			log.Fatalf("[ERROR] Failed to start process: %v", err)
		}
		fmt.Printf("Process '%s' started\n", *startProc)
		sv.PrintStatus()
		return

	case *stopProc != "":
		if err := sv.StopProcess(*stopProc); err != nil {
			log.Fatalf("[ERROR] Failed to stop process: %v", err)
		}
		fmt.Printf("Process '%s' stopped\n", *stopProc)
		sv.PrintStatus()
		return

	case *restartProc != "":
		if err := sv.RestartProcess(*restartProc); err != nil {
			log.Fatalf("[ERROR] Failed to restart process: %v", err)
		}
		fmt.Printf("Process '%s' restarted\n", *restartProc)
		sv.PrintStatus()
		return

	case *runProc != "":
		runProcessForeground(sv, *runProc)
		return
	}

	// Стандартный режим работы
	runSupervisor(sv, tuiMode, cfgPath)
}

func listAllProcesses(cfg *config.Config) {
	fmt.Println("\nConfigured processes:")
	for i, p := range cfg.Processes {
		fmt.Printf("%d. %s\n", i+1, p.Name)
		fmt.Printf("   Command: %s %s\n", p.Command, strings.Join(p.Args, " "))
		fmt.Printf("   Autostart: %v, Autorestart: %s\n", p.Autostart, p.Autorestart)
		fmt.Println()
	}
}

func handleReload(sv *supervisor.Supervisor, cfgPath *string) {
	fmt.Println("[INFO] Reloading configuration...")
	newCfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("[ERROR] Config reload failed: %v", err)
	}
	sv.ReloadConfig(newCfg)
	fmt.Println("[INFO] Configuration reloaded successfully")
	sv.PrintStatus()
}

func runProcessForeground(sv *supervisor.Supervisor, procName string) {
	fmt.Printf("Running process '%s' in foreground...\n", procName)

	// Создаем канал для отслеживания завершения
	done := make(chan struct{})

	// Специальный логгер для foreground режима
	sv.SetLogger(func(log string) {
		if strings.Contains(log, "["+procName+"]") {
			fmt.Println(log)
		}
	})

	// Запускаем процесс
	if err := sv.StartProcess(procName); err != nil {
		log.Fatalf("[ERROR] Failed to start process: %v", err)
	}

	// Обработка сигналов
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Ожидаем завершения или сигнала
	go func() {
		<-sigCh
		fmt.Printf("\nStopping process '%s'...\n", procName)
		sv.StopProcess(procName)
		close(done)
	}()

	// Периодическая проверка состояния
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			fmt.Println("Process stopped")
			return
		case <-ticker.C:
			status := sv.GetProcessStatus(procName)
			if status == process.Stopped || status == process.Failed {
				fmt.Println("Process completed")
				return
			}
		}
	}
}

func runSupervisor(sv *supervisor.Supervisor, tuiMode *bool, cfgPath *string) {
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

		log.Println("Entering daemon mode. Press Ctrl+C to exit.")

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
				sv.PrintStatus()
			}
		}
	}

	// Этот код достижим только если вышли из TUI
	sv.StopAll()
	log.Println("[INFO] Supervisor stopped")
}
