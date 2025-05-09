package pubsub

import (
	"context"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Проверка на успешный запуск сервера
func setupTestServer(t *testing.T) (*grpc.Server, string) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	RegisterPubSubServer(grpcServer, NewServer())

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	return grpcServer, listener.Addr().String()
}

// Проверка на то, что все сообщения получены в правильном порядке.
func TestSubscribeAndPublish(t *testing.T) {
	grpcServer, addr := setupTestServer(t)
	defer grpcServer.GracefulStop()

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := NewPubSubClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Subscribe(ctx, &SubscribeRequest{Key: "test-key"})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	messages := []string{"event1", "event2"}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, msg := range messages {
			_, err := client.Publish(ctx, &PublishRequest{Key: "test-key", Data: msg})
			if err != nil {
				t.Errorf("Publish failed: %v", err)
			}
		}
	}()

	received := make([]string, 0, len(messages))
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < len(messages); i++ {
			event, err := stream.Recv()
			if err != nil {
				t.Errorf("Stream recv failed: %v", err)
				return
			}
			received = append(received, event.Data)
		}
	}()

	wg.Wait()

	if len(received) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(received))
	}
	for i, msg := range messages {
		if received[i] != msg {
			t.Errorf("Expected message %v at index %d, got %v", msg, i, received[i])
		}
	}
}

// Проверка на пустые данные или пустой ключ
func TestPublishInvalidRequest(t *testing.T) {
	grpcServer, addr := setupTestServer(t)
	defer grpcServer.GracefulStop()

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := NewPubSubClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Publish(ctx, &PublishRequest{Key: "", Data: "data"})
	if err == nil {
		t.Error("Expected error for empty key")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", status.Code(err))
	}

	_, err = client.Publish(ctx, &PublishRequest{Key: "key", Data: ""})
	if err == nil {
		t.Error("Expected error for empty data")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", status.Code(err))
	}
}
