/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/google/uuid"
	"github.com/hpe/access-manager/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TraceIDInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// Call the gRPC handler with the modified context.
	return handler(GetTraceIDsFromIncomingCtx(ctx), req)
}

func GetTraceIDsFromIncomingCtx(ctx context.Context) context.Context {
	requestID := ""
	subscriptionID := ""

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		valuesRequestID := md.Get(logger.RequestID)
		if len(valuesRequestID) > 0 {
			requestID = valuesRequestID[0]
		}

		valuesSubID := md.Get(logger.SubscriptionID)
		if len(valuesSubID) > 0 {
			subscriptionID = valuesSubID[0]
		}
	}

	if strings.TrimSpace(requestID) == "" {
		// Generate a UUID for the request.
		requestID = uuid.New().String()
	}

	// Attach the request ID to the context.
	ctx = context.WithValue(ctx, logger.RequestID, requestID) //nolint:staticcheck

	if subscriptionID != "" {
		ctx = context.WithValue(ctx, logger.SubscriptionID, subscriptionID) //nolint:staticcheck
	}

	return ctx
}

func AddRequestIDToContext(ctx context.Context) context.Context {
	// Extract the request ID from the context.
	requestID, ok := ctx.Value(logger.RequestID).(string)
	if !ok {
		return ctx
	}
	md := metadata.Pairs(logger.RequestID, requestID)

	return metadata.NewOutgoingContext(ctx, md)
}

func AddSubscriptionIDToContext(ctx context.Context) context.Context {
	randomSubscriptionTracingID := uuid.New().String()

	subscriptionCtx := context.WithValue(ctx, logger.SubscriptionID, randomSubscriptionTracingID) //nolint:staticcheck

	md := metadata.Pairs(logger.SubscriptionID, randomSubscriptionTracingID)

	return metadata.NewOutgoingContext(subscriptionCtx, md)
}

func AddTraceIDInGateway() runtime.ServeMuxOption {
	return runtime.WithMetadata(func(ctx context.Context, r *http.Request) metadata.MD {
		// Attach the request ID to the context.
		requestID := uuid.New().String()
		ctx = context.WithValue(ctx, logger.RequestID, requestID) //nolint:staticcheck
		logger.GetLogger().Info().Ctx(ctx).Msg(fmt.Sprintf("received request URL: %s, method: %s", r.URL.String(), r.Method))

		md := make(map[string]string)
		md[logger.RequestID] = requestID
		return metadata.New(md)
	})
}
