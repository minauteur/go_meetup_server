package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bufbuild/connect-go"
	greetingpb "github.com/minauteur/go_meetup_api/go/api/greeting/v1"
	greetingpbconnect "github.com/minauteur/go_meetup_api/go/api/greeting/v1/greetingpbconnect"
	waitpb "github.com/minauteur/go_meetup_api/go/api/wait/v1"
	waitpbconnect "github.com/minauteur/go_meetup_api/go/api/wait/v1/waitpbconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// set a timeout
const timeout = time.Second * 20
const addr = "0.0.0.0:8080"

func main() {

	// create a wait group for awaiting inflight requests to complete on shutdown
	wg := sync.WaitGroup{}

	ctx, notifyStop := signal.NotifyContext(context.Background(), syscall.SIGINT)

	// create a channel for listening to signals
	signals := make(chan os.Signal, 1)

	// signal.Notify routes specified signals to the provided channel
	// here we just listen for SIGINT e.g. Ctrl + c
	signal.Notify(
		signals,
		syscall.SIGINT,
	)

	// create a new http request multiplexer
	mux := http.NewServeMux()

	// create a handler in the standard fashion for our greeting server
	gPath, gHandler := greetingpbconnect.NewGreetingAPIHandler(GreetingAPIServer{})
	mux.Handle(gPath, gHandler)

	// for our wait server, which could take some time responding to requests,
	// we can wrap the standard handler in an http.HandlerFunc that includes a waitgroup
	// for waiting on inflight requests to be handled before shutting down.
	wPath, wHandler := waitpbconnect.NewWaitAPIHandler(WaitAPIServer{})
	wh := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		wg.Add(1)
		log.Printf("added to waitgroup")
		defer wg.Done()
		wHandler.ServeHTTP(rw, req)
	})
	mux.Handle(wPath, wh)

	// create a listener for our server
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("listener error: %s", err.Error())
	}
	log.Println("Listening on", addr)

	// create an http server with our request multiplexer and its registered handlers
	srv := http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	// spawn a goroutine for our server
	go func() {
		err := srv.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("fatal server error: %s", err.Error())
		}
		// log after the error is returned to confirm that the goroutine completes its execution
		log.Print("stopped serving...")
	}()

	// await signals, and assign any incoming value to "signal" for logging
	// it could also be used for signal-specific handling
	signal := <-signals

	// create a channel for our workgroup to signal the main thread when it has
	// finished handling inflight requests
	done := make(chan struct{})

	fmt.Println()
	log.Printf("received signal: %v\n", signal)

	// spawn a thread to wait for requests to finish
	go func() {
		log.Printf("waiting for requests to finish...")
		wg.Wait()
		log.Printf("done!")
		done <- struct{}{}
	}()

	// if wg.Wait() returns before the timeout expires, we've handled all inlfight requests successfully.
	// Otherwise, we abort if the timeout has expired.
	// In a real situation we might want to do some additional cleanup or handling
	select {
	case <-done:
		log.Printf("Handled all inflight requests; exiting...")
	case <-time.After(timeout):
		log.Printf("Timeout expired; closing active connections...")
	}

	// attempt to shut down the server,
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %s", err.Error())
		os.Exit(1)
	} else {
		log.Printf("gracefully stopped")
	}
	notifyStop()

	os.Exit(0)

}

type WaitAPIServer struct {
	waitpbconnect.UnimplementedWaitAPIHandler
	waitpbconnect.WaitAPIHandler
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

func (w WaitAPIServer) Wait(ctx context.Context, req *connect.Request[waitpb.WaitRequest]) (*connect.Response[waitpb.WaitResponse], error) {
	msg := req.Msg

	wt := msg.GetWaitTime()

	log.Printf("WaitAPIServer: waiting %d seconds...", wt)
	res := connect.Response[waitpb.WaitResponse]{
		Msg: &waitpb.WaitResponse{
			Message: fmt.Sprintf("waited %d seconds", wt),
		},
	}
	waitCh := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(wt))
		waitCh <- true
	}()
	seconds := 0
	for {
		select {
		case <-time.After(time.Second):
			seconds += 1
		case <-waitCh:
			return &res, nil
		case <-ctx.Done():
			log.Printf("context Error: %s", ctx.Err().Error())
			return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("requested wait: %ds interrupted at %ds", wt, seconds))
		}
	}

}
