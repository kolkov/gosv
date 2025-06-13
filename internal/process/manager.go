package process

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/kolkov/gosv/internal/config"
)

type Status string

const (
	Stopped  Status = "stopped"
	Starting Status = "starting"
	Running  Status = "running"
	Stopping Status = "stopping"
	Failed   Status = "failed"
)

const (
	MaxRestarts         = 5
	InitialRestartDelay = 1 * time.Second
	MaxRestartDelay     = 30 * time.Second
)

type ProcessInfo struct {
	PID       int
	Status    Status
	StartTime time.Time
	Restarts  int
	ExitError error
}

type Process struct {
	ID           string
	Cmd          *exec.Cmd
	Status       Status
	Config       config.ProcessConfig
	restart      bool
	quit         chan struct{}
	mu           sync.Mutex
	startTime    time.Time
	restartCount int
	restartDelay time.Duration
	exitError    error
	logger       func(string) // Функция для логирования
}

type Manager struct {
	processes map[string]*Process
	mu        sync.RWMutex
	logger    func(string) // Общий логгер
}

func NewManager(logger func(string)) *Manager {
	return &Manager{
		processes: make(map[string]*Process),
		logger:    logger,
	}
}

// Добавляем метод для установки логгера
func (m *Manager) SetLogger(logger func(string)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger = logger

	// Обновляем логгеры для всех процессов
	for _, proc := range m.processes {
		proc.mu.Lock()
		proc.logger = logger
		proc.mu.Unlock()
	}
}

func (m *Manager) AddProcess(cfg config.ProcessConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := &Process{
		ID:           cfg.Name,
		Config:       cfg,
		Status:       Stopped,
		quit:         make(chan struct{}),
		restartDelay: InitialRestartDelay,
		logger:       m.logger, // Используем общий логгер
	}

	if cfg.Autorestart == "always" {
		p.restart = true
	}

	m.processes[cfg.Name] = p
}

func (m *Manager) Start(name string) error {
	m.mu.RLock()
	p, exists := m.processes[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process not found: %s", name)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Status != Stopped && p.Status != Failed {
		return fmt.Errorf("process already running: %s", name)
	}

	p.Status = Starting
	p.exitError = nil
	go p.run()

	return nil
}

func (m *Manager) Stop(name string) error {
	m.mu.RLock()
	p, exists := m.processes[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process not found: %s", name)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Status != Running && p.Status != Starting {
		return nil
	}

	p.Status = Stopping
	p.restart = false
	close(p.quit)
	return nil
}

func (m *Manager) Status() map[string]*ProcessInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]*ProcessInfo)
	for name, proc := range m.processes {
		proc.mu.Lock()
		info := &ProcessInfo{
			Status:    proc.Status,
			StartTime: proc.startTime,
			Restarts:  proc.restartCount,
			ExitError: proc.exitError,
		}

		if proc.Cmd != nil && proc.Cmd.Process != nil {
			info.PID = proc.Cmd.Process.Pid
		}

		statuses[name] = info
		proc.mu.Unlock()
	}
	return statuses
}

func (p *Process) run() {
	defer func() {
		p.mu.Lock()
		if p.Status != Stopping {
			p.Status = Stopped
		}
		p.mu.Unlock()
	}()

	for {
		p.mu.Lock()
		p.Status = Starting
		p.startTime = time.Now()
		p.mu.Unlock()

		// Reset restart delay for new runs
		p.restartDelay = InitialRestartDelay

		if p.logger != nil {
			p.logger(fmt.Sprintf("[INFO] Starting process: %s %v", p.Config.Command, p.Config.Args))
		}

		ctx, cancel := context.WithCancel(context.Background())
		cmd := exec.CommandContext(ctx, p.Config.Command, p.Config.Args...)
		cmd.Dir = p.Config.Directory

		// Windows-specific process attributes
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
			HideWindow:    true,
		}

		cmd.Env = os.Environ()
		for k, v := range p.Config.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		// Create output pipes
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		p.mu.Lock()
		p.Cmd = cmd
		p.mu.Unlock()

		// Start the process
		if err := cmd.Start(); err != nil {
			p.mu.Lock()
			p.Status = Failed
			p.exitError = fmt.Errorf("start failed: %w", err)
			p.mu.Unlock()
			if p.logger != nil {
				p.logger(fmt.Sprintf("[ERROR] Process %s failed to start: %v", p.ID, err))
			}
			return
		}

		if p.logger != nil {
			p.logger(fmt.Sprintf("[INFO] Process %s started with PID: %d", p.ID, cmd.Process.Pid))
		}

		p.mu.Lock()
		p.Status = Running
		p.mu.Unlock()

		// Real-time output handling
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				log := fmt.Sprintf("[%s][%d] %s", p.ID, cmd.Process.Pid, scanner.Text())
				if p.logger != nil {
					p.logger(log)
				}
			}
		}()

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				log := fmt.Sprintf("[%s][%d][ERROR] %s", p.ID, cmd.Process.Pid, scanner.Text())
				if p.logger != nil {
					p.logger(log)
				}
			}
		}()

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-p.quit:
			cancel()
			if p.logger != nil {
				p.logger(fmt.Sprintf("[INFO] Stopping process: %s (PID: %d)", p.ID, cmd.Process.Pid))
			}
			if p.Config.StopSignal == "SIGKILL" {
				cmd.Process.Kill()
			} else {
				// Send interrupt signal and wait with timeout
				cmd.Process.Signal(os.Interrupt)
				select {
				case <-done:
					if p.logger != nil {
						p.logger(fmt.Sprintf("[INFO] Process %s stopped gracefully", p.ID))
					}
				case <-time.After(p.Config.StopWait):
					if p.logger != nil {
						p.logger(fmt.Sprintf("[WARN] Force killing process %s after timeout", p.ID))
					}
					cmd.Process.Kill()
					<-done
				}
			}
			<-done
			return

		case err := <-done:
			cancel()
			p.mu.Lock()
			if err != nil {
				p.Status = Failed
				p.exitError = fmt.Errorf("exit error: %w", err)
				if p.logger != nil {
					p.logger(fmt.Sprintf("[ERROR] Process %s (PID: %d) exited with error: %v", p.ID, cmd.Process.Pid, err))
				}
			} else {
				p.Status = Stopped
				if p.logger != nil {
					p.logger(fmt.Sprintf("[INFO] Process %s (PID: %d) exited normally", p.ID, cmd.Process.Pid))
				}
			}
			p.mu.Unlock()
		}

		p.mu.Lock()
		currentRestart := p.restart
		currentRestartCount := p.restartCount
		p.mu.Unlock()

		if !currentRestart {
			return
		}

		// Check restart limits
		if currentRestartCount >= MaxRestarts {
			p.mu.Lock()
			p.Status = Failed
			p.restart = false
			if p.logger != nil {
				p.logger(fmt.Sprintf("[WARN] Process %s reached max restarts (%d), stopping", p.ID, MaxRestarts))
			}
			p.mu.Unlock()
			return
		}

		// Increase restart delay exponentially
		p.restartDelay = time.Duration(float64(p.restartDelay) * 1.5)
		if p.restartDelay > MaxRestartDelay {
			p.restartDelay = MaxRestartDelay
		}

		if p.logger != nil {
			p.logger(fmt.Sprintf("[INFO] Restarting process: %s in %v (attempt %d/%d)",
				p.ID, p.restartDelay.Round(time.Millisecond), currentRestartCount+1, MaxRestarts))
		}

		select {
		case <-p.quit:
			return
		case <-time.After(p.restartDelay):
		}

		p.mu.Lock()
		p.restartCount++
		p.mu.Unlock()
	}
}
