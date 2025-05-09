package pubsub

import (
	"context"

	"github.com/Pulontaine/VK_internship/subpub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	bus subpub.SubPub
	UnimplementedPubSubServer
}

func NewServer() *Server {
	return &Server{
		bus: subpub.NewSubPub(),
	}
}

func (s *Server) Subscribe(req *SubscribeRequest, stream PubSub_SubscribeServer) error {
	if req.Key == "" {
		return status.Error(codes.InvalidArgument, "key cannot be empty")
	}

	eventChan := make(chan interface{})
	defer close(eventChan)

	subscription, err := s.bus.Subscribe(req.Key, func(msg interface{}) {
		if data, ok := msg.(string); ok {
			eventChan <- data
		}
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}
	defer subscription.Unsubscribe()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case data, ok := <-eventChan:
			if !ok {
				return status.Error(codes.Internal, "event channel closed")
			}
			if err := stream.Send(&Event{Data: data.(string)}); err != nil {
				return status.Errorf(codes.Internal, "failed to send event: %v", err)
			}
		}
	}
}

func (s *Server) Publish(ctx context.Context, req *PublishRequest) (*emptypb.Empty, error) {
	if err := s.bus.Publish(req.Key, req.Data); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish: %v", err)
	}
	return &emptypb.Empty{}, nil
}
