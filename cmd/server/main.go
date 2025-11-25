package main

import (
	"context"
	"crypto/tls"
	"fmt"
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
	// Example in-memory storage
	messages map[string][]*chatv1.ChatMessage // room_id → list of messages
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		clients: make(map[chatv1.ChatService_StreamServer]struct{}),
	}
}

func (s *ChatServer) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.SendmessageResponse, error) {
	fmt.Println("RPC SendMessage:: starting")
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
	if s.messages == nil {
		s.messages = make(map[string][]*chatv1.ChatMessage)
	}
	s.messages[msg.RoomId] = append(s.messages[msg.RoomId], msg)

	fmt.Printf(`RPC Sending Message "%s"`+"\n", msg.Text)
	// Return the full message
	return &chatv1.SendmessageResponse{
		Message: msg,
	}, nil
}

// ChatStream handles bidirectional streaming
func (s *ChatServer) Stream(stream chatv1.ChatService_StreamServer) error {
	// Register client
	s.mu.Lock()
	s.clients[stream] = struct{}{}
	s.mu.Unlock()
	log.Println("Client connected to stream")

	incoming := make(chan *chatv1.StreamEvent)
	errs := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Recv goroutine ---
	go func() {
		for {
			event, err := stream.Recv()
			if err != nil {
				st, ok := status.FromError(err)
				if ok {
					switch st.Code() {
					case codes.Canceled, codes.Unavailable:
						log.Println("client stream closed:", st.Message())
						errs <- nil
						return
					}
				}
				log.Println("Error receiving from client:", err)
				errs <- err
				return
			}

			select {
			case incoming <- event:
			case <-ctx.Done():
				log.Println("stream context canceled")
				return
			}
		}
	}()

	for {
		select {
		case evt := <-incoming:
			log.Println("Handling streaming payload")
			switch payload := evt.Payload.(type) {
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
				log.Printf("[control] type=%v room=%s", c.Action, c.RoomId)

			default:
				log.Printf("unknown event payload: %T", payload)
			}
			s.broadcast(evt, stream)
		case err := <-errs:
			if err != nil {
				log.Println("stream loop exited with error:", err)
			}
			return err
		case <-ctx.Done():
			log.Println("WS closed, exiting main loop")
			return nil
		}

	}
}

// broadcast sends event to all clients except the sender
func (s *ChatServer) broadcast(event *chatv1.StreamEvent, sender chatv1.ChatService_StreamServer) {
	log.Println("incoming broadcast request")
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
