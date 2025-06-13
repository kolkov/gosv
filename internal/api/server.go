package api

import (
	"context"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"time"

	"github.com/kolkov/gosv/api/gosv"
	"github.com/kolkov/gosv/internal/service"
	"google.golang.org/grpc"
)

type Server struct {
	gosv.UnimplementedSupervisorServer
	sv service.SupervisorService
}

func NewServer(sv service.SupervisorService) *Server {
	return &Server{sv: sv}
}

func (s *Server) StartProcess(ctx context.Context, req *gosv.ProcessRequest) (*gosv.Response, error) {
	// Перед запуском остановим процесс, если он уже работает
	_ = s.sv.StopProcess(req.Name) // Игнорируем ошибку если процесс не найден
	time.Sleep(100 * time.Millisecond)

	if err := s.sv.StartProcess(req.Name); err != nil {
		return &gosv.Response{Success: false, Message: err.Error()}, nil
	}
	return &gosv.Response{Success: true, Message: "Process started"}, nil
}

func (s *Server) StopProcess(ctx context.Context, req *gosv.ProcessRequest) (*gosv.Response, error) {
	if err := s.sv.StopProcess(req.Name); err != nil {
		return &gosv.Response{Success: false, Message: err.Error()}, nil
	}
	return &gosv.Response{Success: true, Message: "Process stopped"}, nil
}

func (s *Server) RestartProcess(ctx context.Context, req *gosv.ProcessRequest) (*gosv.Response, error) {
	if err := s.sv.RestartProcess(req.Name); err != nil {
		return &gosv.Response{Success: false, Message: err.Error()}, nil
	}
	return &gosv.Response{Success: true, Message: "Process restarted"}, nil
}

func (s *Server) GetStatus(ctx context.Context, req *gosv.StatusRequest) (*gosv.StatusResponse, error) {
	statuses := s.sv.Status()
	resp := &gosv.StatusResponse{
		Processes: make([]*gosv.ProcessStatus, 0, len(statuses)),
	}

	for name, info := range statuses {
		pbStatus := &gosv.ProcessStatus{
			Name:     name,
			Status:   string(info.Status),
			Pid:      int32(info.PID),
			Restarts: int32(info.Restarts),
		}
		if info.ExitError != nil {
			pbStatus.Error = info.ExitError.Error()
		}
		resp.Processes = append(resp.Processes, pbStatus)
	}

	return resp, nil
}

func StartGRPCServer(sv service.SupervisorService, port string) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	gosv.RegisterSupervisorServer(s, NewServer(sv))

	// Включаем рефлексию для использования с grpcurl
	reflection.Register(s)

	log.Printf("gRPC server listening on :%s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
