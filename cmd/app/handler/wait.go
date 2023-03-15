package handler

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
	// get the message from the connect.Request
	msg := req.Msg

	// get the wait time from the message
	wt := msg.GetWaitTime()

	// construct response value
	log.Printf("WaitAPIServer: waiting %d seconds...", wt)
	res := connect.Response[waitpb.WaitResponse]{
		Msg: &waitpb.WaitResponse{
			Message: fmt.Sprintf("waited %d seconds", wt),
		},
	}

	// create a channel for notifying the handler when we're done waiting
	doneWaiting := make(chan bool)
	// wait on a goroutine so that we can count while we wait
	go func() {
		time.Sleep(time.Second * time.Duration(wt))
		close(doneWaiting)
	}()

	// 0 count value
	seconds := 0
	// create a channel for ending the goroutine when we're done counting
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

	// finally, block until either:
	// 		we're done waiting, and a response may be sent (success)
	//		a signal from signal.NotifyContext closes the Done channel, aborting the request (failure)
	select {

	// if the handler context is canceled, end the goroutine we started for counting by closing the countDone channel,
	// and include the count in the error response
	case <-ctx.Done():
		close(countDone)
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("wait aborted after %ds", seconds))
	case <-doneWaiting:
		return &res, nil
	}
}
