package main

import (
	"context"
	"fmt"
	"log"

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
	isAdmin := false
	for key, values := range req.Header() {
		for _, value := range values {
			log.Printf("request header:\tkey: %s,\tvalue: %s", key, value)
			if key == "Authorization" && value == "valid_admin_auth" {
				log.Printf("got admin header")
				isAdmin = true
			}
		}
	}
	res := &recpb.GetRecordResponse{
		Id: "some_id_value",
		Record: &recpb.GetRecordResponse_Record{
			Public:  "public value",
			Private: "private value",
		},
	}
	if !isAdmin {
		fm, err := fieldmaskpb.New(res, "record.public")
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed creating fieldmask"))
		}
		filter, _ := fieldmask_utils.MaskFromProtoFieldMask(fm, mapStructFieldNames)
		fRes := &recpb.GetRecordResponse{}
		err = fieldmask_utils.StructToStruct(filter, res, fRes)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed filtering response"))
		}
		return &connect.Response[recpb.GetRecordResponse]{
			Msg: fRes,
		}, nil

	}
	return &connect.Response[recpb.GetRecordResponse]{Msg: res}, nil
}

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
