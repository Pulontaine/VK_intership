package subpub

import (
	"sync"
	"testing"
	"time"
)

// Проверка на работоспособность подписок: порядок и содержание сообщений
func TestSubscribeAndPublish(t *testing.T) {
	bus := NewSubPub()
	subject := "test-subject"
	var received []interface{}
	var mu sync.Mutex

	handler := func(msg interface{}) {
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
	}
	sub, err := bus.Subscribe(subject, handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	messages := []interface{}{"msg1", 42, "msg3"}
	for _, msg := range messages {
		if err := bus.Publish(subject, msg); err != nil {
			t.Fatalf("Publish failed: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(received) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(received))
	}
	for i, msg := range messages {
		if received[i] != msg {
			t.Errorf("Expected message %v at index %d, got %v", msg, i, received[i])
		}
	}
	mu.Unlock()
	sub.Unsubscribe()
}

// Проверка на работоспособность с двумя подписчиками: оба получили сообщения
func TestMultipleSubscribers(t *testing.T) {
	bus := NewSubPub()
	subject := "test-subject"
	var received1, received2 []interface{}
	var mu1, mu2 sync.Mutex

	handler1 := func(msg interface{}) {
		mu1.Lock()
		received1 = append(received1, msg)
		mu1.Unlock()
	}
	sub1, err := bus.Subscribe(subject, handler1)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	handler2 := func(msg interface{}) {
		mu2.Lock()
		received2 = append(received2, msg)
		mu2.Unlock()
	}
	sub2, err := bus.Subscribe(subject, handler2)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if err := bus.Publish(subject, "test"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu1.Lock()
	if len(received1) != 1 || received1[0] != "test" {
		t.Errorf("Subscriber 1: expected [test], got %v", received1)
	}
	mu1.Unlock()

	mu2.Lock()
	if len(received2) != 1 || received2[0] != "test" {
		t.Errorf("Subscriber 2: expected [test], got %v", received2)
	}
	mu2.Unlock()

	sub1.Unsubscribe()
	sub2.Unsubscribe()
}

// Проверка на то, что Publish не блокируется медленным подписчиком
func TestSlowSubscriber(t *testing.T) {
	bus := NewSubPub()
	subject := "test-subject"
	var fastReceived, slowReceived []interface{}
	var muFast, muSlow sync.Mutex

	fastHandler := func(msg interface{}) {
		muFast.Lock()
		fastReceived = append(fastReceived, msg)
		muFast.Unlock()
	}
	_, err := bus.Subscribe(subject, fastHandler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	slowHandler := func(msg interface{}) {
		time.Sleep(200 * time.Millisecond)
		muSlow.Lock()
		slowReceived = append(slowReceived, msg)
		muSlow.Unlock()
	}
	_, err = bus.Subscribe(subject, slowHandler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	start := time.Now()
	if err := bus.Publish(subject, "test"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	muFast.Lock()
	if len(fastReceived) != 1 || fastReceived[0] != "test" {
		t.Errorf("Fast subscriber: expected [test], got %v", fastReceived)
	}
	muFast.Unlock()

	duration := time.Since(start)
	if duration > 100*time.Millisecond {
		t.Errorf("Publish took too long: %v", duration)
	}
}
