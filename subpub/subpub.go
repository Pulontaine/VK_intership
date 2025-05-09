package subpub

import (
	"context"
	"errors"
	"sync"
)

type MessageHandler func(msg interface{})

type Subscription interface {
	Unsubscribe()
}

type SubPub interface {
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	Publish(subject string, msg interface{}) error
	Close(ctx context.Context) error
}

type subscriber struct {
	handler MessageHandler
	msgChan chan interface{}
	done    chan struct{}
}

type subscription struct {
	sub     *subscriber
	subject string
	bus     *subPubImpl
}

func (s *subscription) Unsubscribe() {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()

	for i, sub := range s.bus.subscribers[s.subject] {
		if sub == s.sub {
			s.bus.subscribers[s.subject] = append(s.bus.subscribers[s.subject][:i], s.bus.subscribers[s.subject][i+1:]...)
			close(sub.done)
			return
		}
	}
}

type subPubImpl struct {
	mu          sync.RWMutex
	subscribers map[string][]*subscriber
	closed      bool
}

func NewSubPub() SubPub {
	return &subPubImpl{
		subscribers: make(map[string][]*subscriber),
	}
}

func (b *subPubImpl) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, errors.New("subpub is closed")
	}

	sub := &subscriber{
		handler: cb,
		msgChan: make(chan interface{}, 100),
		done:    make(chan struct{}),
	}

	go func() {
		for {
			select {
			case msg, ok := <-sub.msgChan:
				if !ok {
					return
				}
				sub.handler(msg)
			case <-sub.done:
				return
			}
		}
	}()

	b.subscribers[subject] = append(b.subscribers[subject], sub)

	return &subscription{
		sub:     sub,
		subject: subject,
		bus:     b,
	}, nil
}

func (b *subPubImpl) Publish(subject string, msg interface{}) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.New("subpub is closed")
	}

	for _, sub := range b.subscribers[subject] {
		select {
		case sub.msgChan <- msg:
		default:
		}
	}
	return nil
}

func (b *subPubImpl) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true

	for _, subs := range b.subscribers {
		for _, sub := range subs {
			close(sub.done)
			close(sub.msgChan)
		}
	}

	b.subscribers = make(map[string][]*subscriber)
	return nil
}
