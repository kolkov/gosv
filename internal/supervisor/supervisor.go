package supervisor

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/kolkov/gosv/internal/config"
	"github.com/kolkov/gosv/internal/process"
	"time"
)

type Supervisor struct {
	manager *process.Manager
	config  *config.Config
}

func New(cfg *config.Config) *Supervisor {
	manager := process.NewManager()
	for _, pcfg := range cfg.Processes {
		manager.AddProcess(pcfg)
	}
	return &Supervisor{
		manager: manager,
		config:  cfg,
	}
}

func (s *Supervisor) StartAll() error {
	for _, pcfg := range s.config.Processes {
		if pcfg.Autostart {
			if err := s.manager.Start(pcfg.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Supervisor) StopAll() {
	for _, pcfg := range s.config.Processes {
		s.manager.Stop(pcfg.Name)
	}
}

func (s *Supervisor) ReloadConfig(newCfg *config.Config) {
	s.StopAll()
	s.config = newCfg

	// Создаем новый менеджер с новыми процессами
	s.manager = process.NewManager()
	for _, pcfg := range newCfg.Processes {
		s.manager.AddProcess(pcfg)
	}

	s.StartAll()
}

func (s *Supervisor) Status() map[string]*process.ProcessInfo {
	return s.manager.Status()
}

func (s *Supervisor) PrintStatus() {
	statuses := s.Status()

	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()

	fmt.Println("\n" + cyan("╔═══════════════════════════════════════╗"))
	fmt.Println(cyan("║") + "        " + magenta("PROCESS SUPERVISOR STATUS") + "       " + cyan("║"))
	fmt.Println(cyan("╠══════════════╦═════════╦══════════╦════════╣"))
	fmt.Println(cyan("║") + "   Process   " + cyan("║") + "  PID   " + cyan("║") + "  Status  " + cyan("║") + " Uptime " + cyan("║"))
	fmt.Println(cyan("╠══════════════╬═════════╬══════════╬════════╣"))

	for name, info := range statuses {
		var statusColor func(a ...interface{}) string
		switch info.Status {
		case process.Running:
			statusColor = green
		case process.Starting, process.Stopping:
			statusColor = yellow
		case process.Failed:
			statusColor = red
		default:
			statusColor = cyan
		}

		uptime := "N/A"
		if !info.StartTime.IsZero() {
			uptime = time.Since(info.StartTime).Round(time.Second).String()
		}

		pidStr := "N/A"
		if info.PID > 0 {
			pidStr = fmt.Sprintf("%d", info.PID)
		}

		fmt.Printf(cyan("║")+" %-12s "+cyan("║")+" %-7s "+cyan("║")+" %-8s "+cyan("║")+" %-6s "+cyan("║\n"),
			name,
			pidStr,
			statusColor(string(info.Status)),
			uptime,
		)
	}

	fmt.Println(cyan("╚══════════════╩═════════╩══════════╩════════╝"))
	fmt.Printf("Total processes: %d | Running: %d | Failed: %d\n\n",
		len(statuses),
		countStatus(statuses, process.Running),
		countStatus(statuses, process.Failed),
	)
}

func countStatus(statuses map[string]*process.ProcessInfo, status process.Status) int {
	count := 0
	for _, info := range statuses {
		if info.Status == status {
			count++
		}
	}
	return count
}
