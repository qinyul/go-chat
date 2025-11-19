package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"connectrpc.com/vanguard/vanguardgrpc"
	"github.com/google/uuid"
	"github.com/qinyul/go-chat/gen/go/chat/chatv1"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ChatServer implements ChatServiceServer
type ChatServer struct {
	chatv1.UnimplementedChatServiceServer

	mu      sync.Mutex
	clients map[chatv1.ChatService_StreamServer]struct{}
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		clients: make(map[chatv1.ChatService_StreamServer]struct{}),
	}
}

func (s *ChatServer) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.SendmessageResponse, error) {
	if req == nil || req.Message == nil {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	msg := req.Message

	if msg.Id == "" {
		msg.Id = uuid.NewString()
	}

	if msg.CreatedAt == nil {
		msg.CreatedAt = timestamppb.Now()
	}

	if msg.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id is required")
	}

	if msg.SenderId == "" {
		return nil, status.Error(codes.InvalidArgument, "sender_id is required")
	}

	// Store in memory (example only)

	fmt.Printf(`RPC Sending Message "%s"`+"\n", msg.Text)
	// Return the full message
	return &chatv1.SendmessageResponse{
		Message: msg,
	}, nil
}

// ChatStream handles bidirectional streaming
func (s *ChatServer) ChatStream(stream chatv1.ChatService_StreamServer) error {
	// Register client
	s.mu.Lock()
	s.clients[stream] = struct{}{}
	s.mu.Unlock()

	defer func() {
		// Unregister client
		s.mu.Lock()
		delete(s.clients, stream)
		s.mu.Unlock()
	}()

	for {
		event, err := stream.Recv()
		if err == io.EOF {
			log.Println("Client dissconected")
			return nil
		}
		if err != nil {
			log.Printf("Error receiving from client: %v", err)
			return err
		}

		switch payload := event.Payload.(type) {
		case *chatv1.StreamEvent_Message:
			msg := payload.Message
			log.Printf("[msg] %s: %s", msg.SenderId, msg.Text)

		case *chatv1.StreamEvent_Typing:
			t := payload.Typing
			log.Printf("[typing] user=%s room=%s is_typing=%v", t.UserId, t.RoomId, t.IsTyping)

		case *chatv1.StreamEvent_Presence:
			p := payload.Presence
			log.Printf("[presence] user=%s online=%v", p.UserId, p.Online)

		case *chatv1.StreamEvent_Control:
			c := payload.Control
			log.Printf("[control] type=%v room=%s", c.Type, c.RoomId)

		default:
			log.Printf("unknown event payload: %T", payload)
		}

		s.broadcast(event, stream)
	}
}

// broadcast sends event to all clients except the sender
func (s *ChatServer) broadcast(event *chatv1.StreamEvent, sender chatv1.ChatService_StreamServer) {
	s.mu.Lock()
	clientsCopy := make([]chatv1.ChatService_StreamServer, 0, len(s.clients))
	for c := range s.clients {
		clientsCopy = append(clientsCopy, c)
	}
	s.mu.Unlock()

	for _, client := range clientsCopy {
		if client == sender {
			continue
		}
		if err := client.Send(event); err != nil {
			log.Printf("Failed to send message, removing client: %v", err)
			// Remove disconnected client
			s.mu.Lock()
			delete(s.clients, client)
			s.mu.Unlock()
		}
	}
}

func main() {
	grpcServer := grpc.NewServer()
	chatSrv := NewChatServer()
	chatv1.RegisterChatServiceServer(grpcServer, chatSrv)
	reflection.Register(grpcServer)

	transcoder, err := vanguardgrpc.NewTranscoder(grpcServer)
	if err != nil {
		log.Fatalf("failed to create Vanguard transcoder: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", transcoder)

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	server := &http.Server{
		Addr:      ":8443",
		Handler:   mux,
		TLSConfig: tlsCfg,
	}

	http2.ConfigureServer(server, &http2.Server{})

	log.Default().Print("Listening on :8443 (HTTPS / HTTP/2)")
	if err := server.ListenAndServeTLS("server.crt", "server.key"); err != nil {
		log.Fatalf("ListenAndServeTLS: %v", err)
	}
}
