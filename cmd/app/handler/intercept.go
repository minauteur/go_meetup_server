package handler

import (
	"context"
	"errors"
	"log"

	"github.com/bufbuild/connect-go"
)

const tokenHeader = "Authorization"

func AuthInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
				// Send a token with client requests.
				req.Header().Set(tokenHeader, "client_auth")
			} else if req.Header().Get(tokenHeader) == "" {
				// Check token in handlers.
				return nil, connect.NewError(
					connect.CodeUnauthenticated,
					errors.New("no auth header provided"),
				)
			}
			log.Printf("found auth header; continuing to handler...")
			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}
