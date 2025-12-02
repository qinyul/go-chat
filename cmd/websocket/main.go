package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding"

	"connectrpc.com/vanguard"
	"connectrpc.com/vanguard/vanguardgrpc"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/qinyul/go-chat/gen/go/chat/chatv1"
)

type WSRequest struct {
	Type    string `json:"type"`
	Message struct {
		RoomID   string `json:"room_id"`
		SenderID string `json:"sender_id"`
		Text     string `json:"text"`
	} `json:"message"`
}

type WSError struct {
	Error string `json:"error"`
}

type WSEnvelope struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Server struct {
	grpcClient chatv1.ChatServiceClient
	upgrader   websocket.Upgrader
}

func main() {
	vCodec := vanguard.JSONCodec{}
	grpcCodec := vanguardgrpc.NewCodec(vCodec)
	encoding.RegisterCodec(grpcCodec)

	// TLS for gRPC connection
	creds, err := credentials.NewClientTLSFromFile("server.crt", "localhost")
	if err != nil {
		log.Fatalf("failed to load TLS cert: %v", err)
	}

	conn, err := grpc.NewClient(
		"dns:///localhost:8443",
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcCodec)),
	)
	if err != nil {
		log.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	s := &Server{
		grpcClient: chatv1.NewChatServiceClient(conn),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/ws", s.handleWS)

	go func() {
		log.Println("Websocket hybrid client listening on :8080")
		http.ListenAndServe(":8080", r)
	}()

	// Catch Ctrl+C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Println("Shutting down...")
}

// ----- WebSocket handler -----
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade error:", err)
		return
	}

	defer conn.Close()
	log.Println("Client connected to Websocket")

	// Create an independent context for the gRPC stream(s).
	// When the HTTP request context is Done (client closed), cancel this stream ctx.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-r.Context().Done()
		cancel()
	}()

	// Open one long-lived gRPC stream for this WS connection.
	stream, err := s.grpcClient.Stream(ctx)

	if err != nil {
		log.Println("failed to open gRPC stream:", err)
		s.sendError(conn, "internal: cannot open backend stream")
		return
	}

	defer stream.CloseSend()

	go func() {
		for {
			event, err := stream.Recv()
			log.Println("Recv payload:", event)
			if err != nil {
				log.Println("gRPC stream closed", err)
				return
			}
			s.sendWS(conn, "StreamEvent", event)
		}
	}()

	interupt := make(chan os.Signal, 1)
	signal.Notify(interupt, os.Interrupt)

	for {
		select {
		case <-interupt:
			log.Println("WS closed by interupt")
			return
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("ws read error:", err)
				return
			}

			var req WSRequest

			if err := json.Unmarshal(msg, &req); err != nil {
				s.sendError(conn, "invalid json")
			}
			fmt.Println("incoming send message payload: ", string(msg))
			s.processWSRequest(conn, req, stream)
		}
	}
}

// ----- WS Request Processor -----
func (s *Server) processWSRequest(conn *websocket.Conn, req WSRequest, stream grpc.BidiStreamingClient[chatv1.StreamEvent, chatv1.StreamEvent]) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	switch req.Type {
	case "SendMessage":
		evt := toProtoStreamEvent(&req)
		if err := stream.Send(evt); err != nil {
			log.Println("stream event send error:", err)
			return
		}

	case "GetMessages":
		var r chatv1.GetMessageRequest
		b, _ := json.Marshal(req.Message)
		json.Unmarshal(b, &r)

		resp, err := s.grpcClient.Getmessages(ctx, &r)
		if err != nil {
			s.sendError(conn, err.Error())
			return
		}

		s.sendWS(conn, "GetMessageResult", resp)

	case "StreamEvent":
		log.Println("WS: Sending stream event")
		evt := toProtoStreamEvent(&req)
		if err := stream.Send(evt); err != nil {
			log.Println("stream event send error:", err)
			return
		}
	}

}

func (s *Server) sendWS(conn *websocket.Conn, msgType string, data interface{}) {
	out := WSEnvelope{
		Type: msgType,
		Data: data,
	}
	b, _ := json.Marshal(out)
	conn.WriteMessage(websocket.TextMessage, b)
}

func (s *Server) sendError(conn *websocket.Conn, msg string) {
	b, _ := json.Marshal(WSError{Error: msg})
	conn.WriteMessage(websocket.TextMessage, b)
}

func toProtoStreamEvent(req *WSRequest) *chatv1.StreamEvent {
	return &chatv1.StreamEvent{
		Type: chatv1.EventType_EVENT_TYPE_MESSAGE,
		Payload: &chatv1.StreamEvent_Message{
			Message: &chatv1.ChatMessage{
				RoomId:   req.Message.RoomID,
				SenderId: req.Message.SenderID,
				Text:     req.Message.Text,
			},
		},
	}
}
