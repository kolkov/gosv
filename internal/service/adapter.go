package service

import "github.com/kolkov/gosv/internal/supervisor"

type supervisorAdapter struct {
	*supervisor.Supervisor
}

func (s *supervisorAdapter) StartProcess(name string) error {
	return s.Supervisor.StartProcess(name)
}

func (s *supervisorAdapter) StopProcess(name string) error {
	return s.Supervisor.StopProcess(name)
}

func (s *supervisorAdapter) RestartProcess(name string) error {
	return s.Supervisor.RestartProcess(name)
}

func (s *supervisorAdapter) Status() map[string]*supervisor.ProcessInfo {
	return s.Supervisor.Status()
}

// AsService преобразует Supervisor в SupervisorService
func AsService(s *supervisor.Supervisor) SupervisorService {
	return &supervisorAdapter{s}
}
