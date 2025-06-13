package supervisor

import (
	"fmt"
	"strings"
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

	// Создаем цветные принтеры
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	magenta := color.New(color.FgMagenta, color.Bold).SprintFunc()

	// Рассчитываем максимальные длины для выравнивания
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

	// Шаблоны форматирования
	nameFormat := fmt.Sprintf("%%-%ds", maxNameLen)
	pidFormat := fmt.Sprintf("%%-%ds", maxPidLen)

	fmt.Println()
	fmt.Println(magenta("PROCESS SUPERVISOR STATUS"))
	fmt.Println(strings.Repeat("-", maxNameLen+maxPidLen+25))

	// Заголовок
	fmt.Printf(
		"%s | %s | %-8s | %-8s\n",
		cyan(fmt.Sprintf(nameFormat, "Process")),
		cyan(fmt.Sprintf(pidFormat, "PID")),
		cyan("Status"),
		cyan("Uptime"),
	)
	fmt.Println(strings.Repeat("-", maxNameLen+maxPidLen+25))

	// Данные процессов
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

		// Выбираем цвет для статуса
		var statusColor func(a ...interface{}) string
		switch info.Status {
		case process.Running:
			statusColor = green
			running++
			active++
		case process.Starting, process.Stopping:
			statusColor = yellow
			active++
		case process.Failed:
			statusColor = red
			failed++
		case process.Stopped:
			statusColor = blue
		default:
			statusColor = cyan
		}

		fmt.Printf(
			"%s | %s | %s | %s\n",
			fmt.Sprintf(nameFormat, name),
			fmt.Sprintf(pidFormat, pidStr),
			statusColor(fmt.Sprintf("%-8s", info.Status)),
			fmt.Sprintf("%-8s", uptime),
		)
	}

	fmt.Println(strings.Repeat("-", maxNameLen+maxPidLen+25))
	fmt.Printf("Processes: %d | ", len(statuses))
	green("Running: %d", running)
	fmt.Print(" | ")
	red("Failed: %d", failed)
	fmt.Print(" | ")
	yellow("Active: %d\n\n", active)
}

func (s *Supervisor) RunTUI() {
	app := tview.NewApplication()

	// Создаем таблицу для статуса процессов
	table := tview.NewTable().
		SetBorders(true).
		SetFixed(1, 1)

	// Настраиваем заголовки
	headerStyle := tcell.Style{}.
		Foreground(tcell.ColorYellow).
		Background(tcell.ColorBlack).
		Bold(true)

	table.SetCell(0, 0, tview.NewTableCell("Process").SetStyle(headerStyle))
	table.SetCell(0, 1, tview.NewTableCell("PID").SetStyle(headerStyle))
	table.SetCell(0, 2, tview.NewTableCell("Status").SetStyle(headerStyle))
	table.SetCell(0, 3, tview.NewTableCell("Uptime").SetStyle(headerStyle))
	table.SetCell(0, 4, tview.NewTableCell("Restarts").SetStyle(headerStyle))

	// Текстовое поле для логов
	logView := tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	logView.SetBorder(true).SetTitle("Logs")

	// Создаем flex-контейнер
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(table, 0, 1, true).
		AddItem(logView, 10, 1, false)

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

			// Цвет статуса
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

			table.SetCell(row, 0, tview.NewTableCell(name))
			table.SetCell(row, 1, tview.NewTableCell(pidStr))
			table.SetCell(row, 2, tview.NewTableCell(string(info.Status)).
				SetTextColor(color))
			table.SetCell(row, 3, tview.NewTableCell(uptime))
			table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%d", info.Restarts)))
			row++
		}

		// Удаляем старые строки
		for i := row; i < table.GetRowCount(); i++ {
			table.RemoveRow(i)
		}
	}

	// Первоначальное обновление
	updateTable()

	// Автообновление
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				app.QueueUpdateDraw(updateTable)
			}
		}
	}()

	// Обработка клавиш
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
		}
		return event
	})

	// Запуск приложения
	if err := app.SetRoot(flex, true).SetFocus(table).Run(); err != nil {
		panic(err)
	}
}

func formatUptime(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	s := (d - m*time.Minute) / time.Second
	return fmt.Sprintf("%02dm%02ds", m, s)
}
