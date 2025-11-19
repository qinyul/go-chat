package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"connectrpc.com/vanguard/vanguardgrpc"
	"github.com/qinyul/go-chat/gen/go/chat/chatv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// ChatServer implements ChatServiceServer
type ChatServer struct {
	chatv1.UnimplementedChatServiceServer

	mu sync.Mutex
	clients map[chatv1.ChatService_StreamServer]struct{}
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		clients: make(map[chatv1.ChatService_StreamServer]struct{}),
	}
}

// ChatStream handles bidirectional streaming
func (s *ChatServer) ChatStream(stream chatv1.ChatService_StreamServer) error {
		// Register client
		s.mu.Lock()
		s.clients[stream] = struct{}{}
		s.mu.Unlock()

		defer func () {
				// Unregister client
			s.mu.Lock()
			delete(s.clients, stream)
			s.mu.Unlock()
		}()

		for {
			event, err := stream.RecvMsg()
			if err == io.EOF {
				log.Println("Client dissconected")
				return  nil
			}
			if err != nil {
				log.Printf("Error receiving from client: %v",err)
				return  err
			}

			log.Printf("[%s] %s",)
			s.broadcast(event,stream)
		}
}

// broadcast sends event to all clients except the sender
func (s *ChatServer) broadcast(event *chatv1.StreamEvent, sender chatv1.ChatService_StreamServer) {
	s.mu.Lock()
	clientsCopy := make([]chatv1.ChatService_StreamServer,0,len(s.clients))
	for c := range s.clients {
		clientsCopy = append(clientsCopy, c)
	}
	s.mu.Unlock()

	for _, client := range clientsCopy {
		if (client == sender) {
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
	chatv1.RegisterChatServiceServer(grpcServer,chatSrv)
	reflection.Register(grpcServer)

	transcoder, err := vanguardgrpc.NewTranscoder(grpcServer)
	if err != nil {
		log.Fatalf("failed to create Vanguard transcoder: %v",err)
	}

	mux := http.NewServeMux()
	mux.Handle("/",transcoder)

	lis,err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v",err)
	}
	log.Println("Server listening on :50051 (with Vanguard)")
	if err := http.Serve(lis,mux); err != nil {
		log.Fatalf("http serve failed: %v",err)
	}
}