package handler

import (
	"context"
	"strings"

	"github.com/bufbuild/connect-go"
	greetingpb "github.com/minauteur/go_meetup_api/go/api/greeting/v1"
	greetingpbconnect "github.com/minauteur/go_meetup_api/go/api/greeting/v1/greetingpbconnect"
)

type GreetingAPIServer struct {
	greetingpbconnect.UnimplementedGreetingAPIHandler
	greetingpbconnect.GreetingAPIHandler
}

// basic handler example that responds to requests based on information included in the request data
func (g GreetingAPIServer) Greet(ctx context.Context, req *connect.Request[greetingpb.GreetingMessage]) (*connect.Response[greetingpb.GreetingResponse], error) {
	msg := req.Msg
	res := connect.Response[greetingpb.GreetingResponse]{}

	// if no name or entity type is specified in the request, respond accordingly
	if msg.GetName() == "" && msg.GetEntityType() == greetingpb.GreetingMessage_ENTITY_TYPE_UNKNOWN {
		resMsg := greetingpb.GreetingResponse{
			Message: "Greetings, mysterious being...",
		}
		res.Msg = &resMsg
		return &res, nil
	}

	// otherwise, respond with a greeting based on information we receive in the request
	greetingResponse := "Greetings, "
	if strings.TrimSpace(msg.GetName()) != "" {
		greetingResponse += msg.GetName() + ", "
	}

	// switch on the entity type
	// demonstrating that ENTITY_TYPE_UNKNOWN is our default value
	switch msg.GetEntityType() {
	case greetingpb.GreetingMessage_ENTITY_TYPE_HUMAN:
		greetingResponse += "earthling"
	case greetingpb.GreetingMessage_ENTITY_TYPE_EXTRA_TERRESTRIAL:
		greetingResponse += "spaceling"
	case greetingpb.GreetingMessage_ENTITY_TYPE_UNKNOWN:
		greetingResponse += "being of unknown origin"
	}

	res.Msg = &greetingpb.GreetingResponse{
		Message: greetingResponse,
	}
	return &res, nil

}
