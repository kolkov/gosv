package cmd

import (
	"context"
	"fmt"
	"github.com/kolkov/gosv/api/gosv"
	"log"

	"google.golang.org/grpc"
)

func getClient() (gosv.SupervisorClient, *grpc.ClientConn) {
	// Конфигурируемый адрес (можно вынести в флаги)
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	return gosv.NewSupervisorClient(conn), conn
}

func startProcessRemote(name string) {
	client, conn := getClient()
	defer conn.Close()

	resp, err := client.StartProcess(context.Background(), &gosv.ProcessRequest{Name: name})
	if err != nil {
		log.Fatalf("gRPC error: %v", err)
	}

	if resp.Success {
		fmt.Printf("Process '%s' started via gRPC\n", name)
	} else {
		fmt.Printf("Error: %s\n", resp.Message)
	}
}
