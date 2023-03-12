package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bufbuild/connect-go"
	greetingpb "github.com/minauteur/go_meetup_api/go/api/greeting/v1"
	greetingpbconnect "github.com/minauteur/go_meetup_api/go/api/greeting/v1/greetingpbconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const addr = "0.0.0.0:8080"

func main() {
	mux := http.NewServeMux()
	path, handler := greetingpbconnect.NewGreetingAPIHandler(GreetingAPIServer{})
	mux.Handle(path, handler)
	fmt.Println("... Listening on", addr)
	http.ListenAndServe(
		addr,
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
}

type GreetingAPIServer struct {
	greetingpbconnect.UnimplementedGreetingAPIHandler
	greetingpbconnect.GreetingAPIHandler
}

func (g GreetingAPIServer) Greet(ctx context.Context, req *connect.Request[greetingpb.GreetingMessage]) (*connect.Response[greetingpb.GreetingResponse], error) {
	msg := greetingpb.GreetingResponse{
		Message: "hey",
	}
	res := connect.Response[greetingpb.GreetingResponse]{Msg: &msg}
	return &res, nil
}
