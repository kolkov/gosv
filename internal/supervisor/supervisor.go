package supervisor

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/gdamore/tcell/v2"
	"github.com/kolkov/gosv/internal/config"
	"github.com/kolkov/gosv/internal/process"
	"github.com/rivo/tview"
)

type Supervisor struct {
	manager *process.Manager
	config  *config.Config
	logs    []string
	logMu   sync.Mutex
}

func New(cfg *config.Config) *Supervisor {
	// Создаем временный логгер
	tempLogger := func(log string) {}

	manager := process.NewManager(tempLogger)
	for _, pcfg := range cfg.Processes {
		manager.AddProcess(pcfg)
	}

	return &Supervisor{
		manager: manager,
		config:  cfg,
		logs:    make([]string, 0),
	}
}

// Устанавливаем логгер для менеджера процессов
func (s *Supervisor) SetLogger(logger func(string)) {
	s.manager.SetLogger(logger)
}

// Добавляем лог-сообщение с защитой от гонок
func (s *Supervisor) AddLog(log string) {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	// Ограничиваем количество хранимых логов
	if len(s.logs) > 1000 {
		s.logs = s.logs[1:]
	}
	s.logs = append(s.logs, log)
}

func (s *Supervisor) StartAll() error {
	return s.manager.StartAll()
}

func (s *Supervisor) StartProcess(name string) error {
	return s.manager.Start(name)
}

func (s *Supervisor) StopAll() {
	s.manager.StopAll()
}

func (s *Supervisor) StopProcess(name string) error {
	return s.manager.Stop(name)
}

func (s *Supervisor) RestartProcess(name string) error {
	if err := s.manager.Stop(name); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond) // Краткая задержка
	return s.manager.Start(name)
}

func (s *Supervisor) ReloadConfig(newCfg *config.Config) {
	s.StopAll()
	s.config = newCfg
	s.manager = process.NewManager(s.AddLog)
	for _, pcfg := range newCfg.Processes {
		s.manager.AddProcess(pcfg)
	}
	s.StartAll()
}

func (s *Supervisor) Status() map[string]*process.ProcessInfo {
	return s.manager.Status()
}

func (s *Supervisor) GetProcessStatus(name string) process.Status {
	statuses := s.manager.Status()
	if info, ok := statuses[name]; ok {
		return info.Status
	}
	return process.Stopped
}

func (s *Supervisor) PrintStatus() {
	statuses := s.Status()

	// Create colored printers
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	magenta := color.New(color.FgMagenta, color.Bold).SprintFunc()

	// Calculate max lengths for alignment
	maxNameLen := 8
	maxPidLen := 3
	for name, info := range statuses {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
		if info.PID > 0 {
			pidStr := fmt.Sprintf("%d", info.PID)
			if len(pidStr) > maxPidLen {
				maxPidLen = len(pidStr)
			}
		}
	}

	// Format templates
	nameFormat := fmt.Sprintf("%%-%ds", maxNameLen)
	pidFormat := fmt.Sprintf("%%-%ds", maxPidLen)

	// Add current time to header
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Println()
	fmt.Println(magenta("PROCESS SUPERVISOR STATUS - " + currentTime))
	fmt.Println(strings.Repeat("-", maxNameLen+maxPidLen+35))

	// Header with Restarts column
	fmt.Printf(
		"%s | %s | %-8s | %-8s | %-7s\n",
		cyan(fmt.Sprintf(nameFormat, "Process")),
		cyan(fmt.Sprintf(pidFormat, "PID")),
		cyan("Status"),
		cyan("Uptime"),
		cyan("Restarts"),
	)
	fmt.Println(strings.Repeat("-", maxNameLen+maxPidLen+35))

	// Process data
	running := 0
	failed := 0
	active := 0

	for name, info := range statuses {
		pidStr := "N/A"
		if info.PID > 0 {
			pidStr = fmt.Sprintf("%d", info.PID)
		}

		uptime := "N/A"
		if !info.StartTime.IsZero() {
			uptime = formatUptime(time.Since(info.StartTime))
		}

		// Create colored status string
		var statusStr string
		switch info.Status {
		case process.Running:
			statusStr = green(fmt.Sprintf("%-8s", info.Status))
			running++
			active++
		case process.Starting, process.Stopping:
			statusStr = yellow(fmt.Sprintf("%-8s", info.Status))
			active++
		case process.Failed:
			statusStr = red(fmt.Sprintf("%-8s", info.Status))
			failed++
		case process.Stopped:
			statusStr = blue(fmt.Sprintf("%-8s", info.Status))
		default:
			statusStr = cyan(fmt.Sprintf("%-8s", info.Status))
		}

		// Highlight restarts when near limit
		restarts := fmt.Sprintf("%d", info.Restarts)
		if info.Restarts >= process.MaxRestarts-1 {
			restarts = yellow(restarts)
		} else if info.Restarts > 0 {
			restarts = cyan(restarts)
		}

		fmt.Printf(
			"%s | %s | %s | %s | %s\n",
			fmt.Sprintf(nameFormat, name),
			fmt.Sprintf(pidFormat, pidStr),
			statusStr, // Используем цветную строку статуса
			fmt.Sprintf("%-8s", uptime),
			fmt.Sprintf("%-7s", restarts),
		)

		// Show error details for failed processes
		if info.Status == process.Failed && info.ExitError != nil {
			fmt.Printf("  └─ %s\n", red(info.ExitError.Error()))
		}
	}

	fmt.Println(strings.Repeat("-", maxNameLen+maxPidLen+35))
	fmt.Printf("Processes: %d | ", len(statuses))
	green("Running: %d", running)
	fmt.Print(" | ")
	red("Failed: %d", failed)
	fmt.Print(" | ")
	yellow("Active: %d", active)
	fmt.Print(" | ")
	cyan("Max restarts: %d\n\n", process.MaxRestarts)
}

func (s *Supervisor) RunTUI() {
	app := tview.NewApplication()

	// Устанавливаем логгер для TUI
	s.SetLogger(s.AddLog)

	// Create process status table
	table := tview.NewTable().
		SetBorders(true).
		SetFixed(1, 1)

	// Configure headers
	headerStyle := tcell.Style{}.
		Foreground(tcell.ColorYellow).
		Background(tcell.ColorBlack).
		Bold(true)

	table.SetCell(0, 0, tview.NewTableCell("Process").SetStyle(headerStyle))
	table.SetCell(0, 1, tview.NewTableCell("PID").SetStyle(headerStyle))
	table.SetCell(0, 2, tview.NewTableCell("Status").SetStyle(headerStyle))
	table.SetCell(0, 3, tview.NewTableCell("Uptime").SetStyle(headerStyle))
	table.SetCell(0, 4, tview.NewTableCell("Restarts").SetStyle(headerStyle))

	// Текстовое поле для логов с буферизацией
	logView := tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	logView.SetBorder(true).SetTitle("Logs")
	logView.SetScrollable(true)

	// Создаем flex-контейнер с правильными пропорциями
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(table, 0, 3, true). // 3/4 экрана для таблицы
		AddItem(logView, 0, 1, false) // 1/4 экрана для логов

	// Функция обновления таблицы
	updateTable := func() {
		statuses := s.Status()
		row := 1
		for name, info := range statuses {
			pidStr := "N/A"
			if info.PID > 0 {
				pidStr = fmt.Sprintf("%d", info.PID)
			}

			uptime := "N/A"
			if !info.StartTime.IsZero() {
				uptime = formatUptime(time.Since(info.StartTime))
			}

			// Status color
			var color tcell.Color
			switch info.Status {
			case process.Running:
				color = tcell.ColorGreen
			case process.Starting, process.Stopping:
				color = tcell.ColorYellow
			case process.Failed:
				color = tcell.ColorRed
			case process.Stopped:
				color = tcell.ColorBlue
			default:
				color = tcell.ColorWhite
			}

			// Restarts cell color
			restartColor := tcell.ColorWhite
			if info.Restarts >= process.MaxRestarts-1 {
				restartColor = tcell.ColorYellow
			} else if info.Restarts > 0 {
				restartColor = tcell.Color(6) // Cyan color
			}

			table.SetCell(row, 0, tview.NewTableCell(name))
			table.SetCell(row, 1, tview.NewTableCell(pidStr))
			table.SetCell(row, 2, tview.NewTableCell(string(info.Status)).
				SetTextColor(color))
			table.SetCell(row, 3, tview.NewTableCell(uptime))
			table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%d", info.Restarts)).
				SetTextColor(restartColor))
			row++
		}

		// Remove old rows
		for i := row; i < table.GetRowCount(); i++ {
			table.RemoveRow(i)
		}
	}

	// Функция обновления логов
	updateLogs := func() {
		s.logMu.Lock()
		defer s.logMu.Unlock()

		logView.Clear()
		for _, log := range s.logs {
			fmt.Fprintln(logView, log)
		}

		// Автопрокрутка к концу
		logView.ScrollToEnd()
	}

	// Первоначальное обновление
	updateTable()
	updateLogs()

	// Автообновление
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				app.QueueUpdateDraw(func() {
					updateTable()
					updateLogs()
				})
			}
		}
	}()

	// Key handling
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyTab:
			if app.GetFocus() == table {
				app.SetFocus(logView)
			} else {
				app.SetFocus(table)
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'r', 'R':
				// Handle process restart
				row, _ := table.GetSelection()
				if row > 0 {
					cell := table.GetCell(row, 0)
					if cell != nil {
						processName := cell.Text
						go func() {
							s.manager.Stop(processName)
							time.Sleep(100 * time.Millisecond)
							s.manager.Start(processName)
						}()
					}
				}
			}
		}
		return event
	})

	// Start application
	if err := app.SetRoot(flex, true).SetFocus(table).Run(); err != nil {
		panic(err)
	}
}

func formatUptime(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02dh%02dm", h, m)
	}
	return fmt.Sprintf("%02dm%02ds", m, s)
}
