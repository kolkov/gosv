package service

import "github.com/kolkov/gosv/internal/supervisor"

type SupervisorService interface {
	StartProcess(name string) error
	StopProcess(name string) error
	RestartProcess(name string) error
	Status() map[string]*supervisor.ProcessInfo
}
