package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bufbuild/connect-go"
	greetingpbconnect "github.com/minauteur/go_meetup_api/go/api/greeting/v1/greetingpbconnect"
	recpbconnect "github.com/minauteur/go_meetup_api/go/api/record/v1/recpbconnect"
	waitpbconnect "github.com/minauteur/go_meetup_api/go/api/wait/v1/waitpbconnect"
	handler "github.com/minauteur/go_meetup_server/cmd/app/handler"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const addr = "0.0.0.0:8080"

func main() {

	// use NotifyContext for propagating signals to handlers
	ctx, stopNotify := signal.NotifyContext(context.Background(), syscall.SIGUSR1)

	// create a wait group for awaiting inflight requests to complete on shutdown
	wg := sync.WaitGroup{}

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

	// create a standard connect handler for our greeting api (basic example)
	gPath, gHandler := greetingpbconnect.NewGreetingAPIHandler(handler.GreetingAPIServer{})
	mux.Handle(gPath, gHandler)

	// create a handler with an interceptor for our record api (fieldmask example)
	interceptor := connect.WithInterceptors(handler.AuthInterceptor())
	rPath, rHandler := recpbconnect.NewRecordAPIHandler(handler.RecordAPIServer{}, interceptor)
	mux.Handle(rPath, rHandler)

	// for our wait server, which could take some time responding to requests,
	// we can wrap the standard handler in an http.HandlerFunc that includes a waitgroup
	// for waiting on inflight requests to be handled before shutting down.
	wPath, wHandler := waitpbconnect.NewWaitAPIHandler(handler.WaitAPIServer{})
	wh := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		wg.Add(1)
		log.Printf("added to waitgroup")
		defer wg.Done()
		wHandler.ServeHTTP(rw, req)
	})
	mux.Handle(wPath, wh)

	// create an http server with our request multiplexer and its registered handlers,
	// using the context we created earlier for propagating signals as the server's base context
	srv := http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
		BaseContext: func(l net.Listener) context.Context {
			return context.WithValue(ctx, "isBaseContext", true)
		},
	}

	// spawn a goroutine for our server
	go func() {
		log.Printf("now serving: %s", srv.Addr)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("fatal server error: %s", err.Error())
		}
		// log after the error is returned to confirm that the goroutine completes its execution
		log.Print("stopped serving...")
	}()

	// await signals
	// fine to read because we don't want a SIGINT to exit the process
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

	// set a timeout
	timeout := time.Second * 5

	// if wg.Wait() returns before the timeout expires, we've handled all inflight requests successfully
	// otherwise we close any connections that remain open prematurely if necessary by closing the context we pass to the handler
	// In a real situation we might want to do some additional cleanup or handling
L:
	for {
		select {
		case <-done:
			log.Printf("Handled all inflight requests; exiting...")
			break L

		// this is a quick and dirty way to create a timeout--not for real-world usage
		case <-time.After(timeout):

			log.Printf("Timeout expired; closing active connections...")

			// this will close the Done() channel of our handler without exiting the program,
			// triggering error errors with connect.CodeAborted in responses to any inflight requests
			syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)

			// after the signal has been propagated to the handler, we can cancel the parent context safely
			// according to the documentation for NotifyContext:
			//     The stop function releases resources associated with it,
			//     so code should call stop as soon as the operations running in this Context complete and
			//     signals no longer need to be diverted to the context
			stopNotify()
		}
	}

	// standard shutdown behavior--now that we've handled inflight requests and closed open connections
	// shutdown should be graceful
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %s", err.Error())
	} else {
		log.Printf("gracefully shut down")
	}
}
