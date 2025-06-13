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

// Новая структура для информации о процессе
type ProcessInfo struct {
	PID       int
	Status    Status
	StartTime time.Time
	Restarts  int
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
}

type Manager struct {
	processes map[string]*Process
	mu        sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*Process),
	}
}

func (m *Manager) AddProcess(cfg config.ProcessConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := &Process{
		ID:     cfg.Name,
		Config: cfg,
		Status: Stopped,
		quit:   make(chan struct{}),
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

	if p.Status != Running {
		return nil
	}

	p.Status = Stopping
	p.restart = false
	close(p.quit)
	return nil
}

// Исправленный метод Status
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
		fmt.Printf("[INFO] Process %s stopped\n", p.ID)
		p.mu.Unlock()
	}()

	for {
		p.mu.Lock()
		p.Status = Starting
		p.restartCount++
		p.startTime = time.Now()
		p.mu.Unlock()

		fmt.Printf("[INFO] Starting process: %s %v\n", p.Config.Command, p.Config.Args)

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

		// Создаем каналы для вывода
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		p.mu.Lock()
		p.Cmd = cmd
		p.mu.Unlock()

		// Запускаем процесс
		if err := cmd.Start(); err != nil {
			p.mu.Lock()
			p.Status = Failed
			p.mu.Unlock()
			fmt.Printf("[ERROR] Process %s failed to start: %v\n", p.ID, err)
			return
		}

		fmt.Printf("[INFO] Process %s started with PID: %d\n", p.ID, cmd.Process.Pid)

		p.mu.Lock()
		p.Status = Running
		p.mu.Unlock()

		// Чтение вывода в реальном времени
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				fmt.Printf("[%s][%d] %s\n", p.ID, cmd.Process.Pid, scanner.Text())
			}
		}()

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				fmt.Printf("[%s][%d][ERROR] %s\n", p.ID, cmd.Process.Pid, scanner.Text())
			}
		}()

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-p.quit:
			cancel()
			fmt.Printf("[INFO] Stopping process: %s (PID: %d)\n", p.ID, cmd.Process.Pid)
			if p.Config.StopSignal == "SIGKILL" {
				cmd.Process.Kill()
			} else {
				cmd.Process.Signal(os.Interrupt)
			}
			<-done
			return

		case err := <-done:
			cancel()
			if err != nil {
				p.mu.Lock()
				p.Status = Failed
				p.mu.Unlock()
				fmt.Printf("[ERROR] Process %s (PID: %d) exited with error: %v\n", p.ID, cmd.Process.Pid, err)
			} else {
				p.mu.Lock()
				p.Status = Stopped
				p.mu.Unlock()
				fmt.Printf("[INFO] Process %s (PID: %d) exited normally\n", p.ID, cmd.Process.Pid)
			}
		}

		if !p.restart {
			return
		}

		fmt.Printf("[INFO] Restarting process: %s\n", p.ID)
		select {
		case <-p.quit:
			return
		case <-time.After(1 * time.Second):
		}
	}
}
