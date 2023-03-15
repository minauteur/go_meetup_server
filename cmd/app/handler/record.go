package handler

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bufbuild/connect-go"
	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	recpb "github.com/minauteur/go_meetup_api/go/api/record/v1"
	"github.com/minauteur/go_meetup_api/go/api/record/v1/recpbconnect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type RecordAPIServer struct {
	recpbconnect.UnimplementedRecordAPIHandler
	recpbconnect.RecordAPIHandler
}

func (r RecordAPIServer) Get(ctx context.Context, req *connect.Request[recpb.GetRecordRequest]) (*connect.Response[recpb.GetRecordResponse], error) {
	// mock example of checking request headers in a connect handler
	isAdmin := false
	keys := make([]string, 0)
	for key, values := range req.Header() {
		keys = append(keys, key)
		for _, value := range values {
			if key == "Authorization" && value == "valid" {
				log.Printf("got admin header")
				isAdmin = true
			}
		}
	}
	log.Printf("headers: %s", strings.Join(keys, ","))

	// mock/example response value to demonstrate fieldmask behavior
	res := &recpb.GetRecordResponse{
		Id: "some_id_value",
		Record: &recpb.GetRecordResponse_Record{
			Public:  "public value",
			Private: "private value",
		},
	}

	// if isAdmin is false, then we want to make sure only public fields are visible in the response
	if !isAdmin {

		// create a new fieldmask
		fm, err := fieldmaskpb.New(res, "record.public")
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed creating fieldmask"))
		}

		// create an example filter from the fieldmask
		filter, _ := fieldmask_utils.MaskFromProtoFieldMask(fm, mapStructFieldNames)

		// empty response "destination" for filter results
		fRes := &recpb.GetRecordResponse{}

		// filter response values into destination
		err = fieldmask_utils.StructToStruct(filter, res, fRes)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed filtering response"))
		}

		// return response with only public field value visible
		return &connect.Response[recpb.GetRecordResponse]{
			Msg: fRes,
		}, nil

	}

	// if isAdmin is true, then send the whole unfiltered response
	return &connect.Response[recpb.GetRecordResponse]{Msg: res}, nil
}

// helper function for fieldmask_utils demo
func mapStructFieldNames(s string) string {
	switch s {
	case "record":
		return "Record"
	case "private":
		return "Private"
	case "public":
		return "Public"
	default:
		return ""
	}
}
