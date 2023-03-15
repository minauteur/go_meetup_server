package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bufbuild/connect-go"
	waitpb "github.com/minauteur/go_meetup_api/go/api/wait/v1"
	waitpbconnect "github.com/minauteur/go_meetup_api/go/api/wait/v1/waitpbconnect"
)

type WaitAPIServer struct {
	waitpbconnect.UnimplementedWaitAPIHandler
	waitpbconnect.WaitAPIHandler
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
	doneWaiting := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(wt))
		close(doneWaiting)
	}()
	seconds := 0
	countDone := make(chan bool)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				seconds += 1
			case <-countDone:
				return
			}
		}
	}()
	select {
	case <-ctx.Done():
		close(countDone)
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("wait aborted after %ds", seconds))
	case <-doneWaiting:
		return &res, nil
	}
}
