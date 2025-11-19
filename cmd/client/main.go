package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"connectrpc.com/vanguard"
	"connectrpc.com/vanguard/vanguardgrpc"
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/qinyul/go-chat/gen/go/chat/chatv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding"
)

type Server struct {
	grpcClient chatv1.ChatServiceClient
}

func main() {

	// Use Vanguard JSON codec (you can change to ProtoCodec if you want)
	vCodec := vanguard.JSONCodec{}

	// Wrap it as a gRPC codec
	grpcCodec := vanguardgrpc.NewCodec(vCodec)

	encoding.RegisterCodec(grpcCodec)

	creds, err := credentials.NewClientTLSFromFile("server.crt", "localhost")
	if err != nil {
		log.Fatalf("failed to load TLS cert: %v", err)
	}
	conn, err := grpc.NewClient(
		"dns:///localhost:8443",
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(
			grpc.ForceCodec(grpcCodec),
		),
	)
	if err != nil {
		log.Fatalf("dial error: %v", err)
	}

	defer conn.Close()

	grpcClient := chatv1.NewChatServiceClient(conn)
	s := &Server{grpcClient: grpcClient}

	// ----- CHI ROUTER -----
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/messages", s.handleSendMessage)
	r.Get("/messages", s.handleGetMessages)

	log.Println("REST hybird client listening on :8080")
	http.ListenAndServe(":8080", r)
}

// POST /messages → forwards to RPC SendMessage
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req chatv1.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	fmt.Printf(`REST Sending Message Request to rpc "%s"`+"\n", req.Message.Text)

	resp, err := s.grpcClient.SendMessage(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// GET /messages?room_id=abc&limit=20 → forwards to RPC
func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room_id")
	limitStr := r.URL.Query().Get("limit")

	var limit int32 = 20
	if limitStr != "" {
		fmt.Sscan(limitStr, &limit)
	}

	req := &chatv1.GetMessageRequest{
		RoomId: roomID,
		Limit:  limit,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := s.grpcClient.Getmessages(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}
