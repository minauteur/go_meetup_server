package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bufbuild/connect-go"
	greetingpb "github.com/minauteur/go_meetup_api/go/api/greeting/v1"
	greetingpbconnect "github.com/minauteur/go_meetup_api/go/api/greeting/v1/greetingpbconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const addr = "0.0.0.0:8080"

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(
		signals,
		syscall.SIGINT, // Ctrl+c
	)
	mux := http.NewServeMux()
	path, handler := greetingpbconnect.NewGreetingAPIHandler(GreetingAPIServer{})
	mux.Handle(path, handler)
	srv := http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("listener error: %s", err.Error())
	}
	log.Println("Listening on", addr)

	// spawn a goroutine for our server
	go func() {
		if err := srv.Serve(ln); err == http.ErrServerClosed {
			log.Printf("server closing...")
		}
	}()

	// await signals, and assign any incoming value to "signal" for logging
	// it could also be used for signal-specific handling
	signal := <-signals
	log.Printf("\nreceived signal: %v\n", signal)
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %s", err.Error())
	} else {
		log.Printf("gracefully stopped")
		os.Exit(0)
	}
}

type GreetingAPIServer struct {
	greetingpbconnect.UnimplementedGreetingAPIHandler
	greetingpbconnect.GreetingAPIHandler
}

func (g GreetingAPIServer) Greet(ctx context.Context, req *connect.Request[greetingpb.GreetingMessage]) (*connect.Response[greetingpb.GreetingResponse], error) {
	msg := req.Msg
	res := connect.Response[greetingpb.GreetingResponse]{}
	if msg.GetName() == "" && msg.GetEntityType() == greetingpb.GreetingMessage_ENTITY_TYPE_UNKNOWN {
		resMsg := greetingpb.GreetingResponse{
			Message: "Greetings, mysterious being...",
		}
		res.Msg = &resMsg
		return &res, nil
	}
	greetingResponse := "Greetings, "
	if strings.TrimSpace(msg.GetName()) != "" {
		greetingResponse += msg.GetName() + ", "
	}
	switch msg.GetEntityType() {
	case greetingpb.GreetingMessage_ENTITY_TYPE_HUMAN:
		greetingResponse += "earthling"
	case greetingpb.GreetingMessage_ENTITY_TYPE_EXTRA_TERRESTRIAL:
		greetingResponse += "spaceling"
	default:
		greetingResponse += "being of unknown origin"
	}

	res.Msg = &greetingpb.GreetingResponse{
		Message: greetingResponse,
	}
	return &res, nil

}
