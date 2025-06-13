package supervisor

import (
	"fmt"
	"sync"
	"time"

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
func (s *Supervisor) setLogger() {
	s.manager.SetLogger(s.AddLog)
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
	s.manager = process.NewManager(s.AddLog)
	for _, pcfg := range newCfg.Processes {
		s.manager.AddProcess(pcfg)
	}
	s.StartAll()
}

func (s *Supervisor) Status() map[string]*process.ProcessInfo {
	return s.manager.Status()
}

func (s *Supervisor) PrintStatus() {
	// ... (без изменений) ...
}

func (s *Supervisor) RunTUI() {
	app := tview.NewApplication()

	// Устанавливаем логгер, который добавляет сообщения в буфер
	s.setLogger()

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
