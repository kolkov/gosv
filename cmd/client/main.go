package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kolkov/gosv/api/gosv"
	"google.golang.org/grpc"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: client.exe <server:port> <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  status   - get processes status")
		fmt.Println("  start <name> - start process")
		fmt.Println("  stop <name>  - stop process")
		return
	}

	conn, err := grpc.Dial(os.Args[1], grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := gosv.NewSupervisorClient(conn)

	switch os.Args[2] {
	case "status":
		resp, err := client.GetStatus(context.Background(), &gosv.StatusRequest{})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Processes status:")
		for _, proc := range resp.Processes {
			// Исправляем форматирование - добавляем закрывающую скобку
			fmt.Printf("- %s: %s (PID: %d, restarts: %d)",
				proc.Name, proc.Status, proc.Pid, proc.Restarts)
			if proc.Error != "" {
				fmt.Printf(", error: %s", proc.Error)
			}
			fmt.Println() // Добавляем перевод строки
		}

	case "start":
		if len(os.Args) < 4 {
			log.Fatal("Missing process name")
		}
		resp, err := client.StartProcess(context.Background(), &gosv.ProcessRequest{Name: os.Args[3]})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Start response: success=%v, message=%s\n", resp.Success, resp.Message)

	case "stop":
		if len(os.Args) < 4 {
			log.Fatal("Missing process name")
		}
		resp, err := client.StopProcess(context.Background(), &gosv.ProcessRequest{Name: os.Args[3]})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Stop response: success=%v, message=%s\n", resp.Success, resp.Message)

	default:
		log.Fatalf("Unknown command: %s", os.Args[2])
	}
}
