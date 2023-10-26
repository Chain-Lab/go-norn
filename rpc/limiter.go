/**
  @author: decision
  @date: 2023/10/23
  @note:
**/

package rpc

import (
	"context"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RateLimiter struct {
	rate  *rate.Limiter
	limit int
}

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		rate:  rate.NewLimiter(rate.Limit(limit), limit),
		limit: limit,
	}
}

func (r *RateLimiter) UnaryInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if !r.rate.Allow() {
		return nil, status.Errorf(codes.ResourceExhausted, "Too many requests")
	}

	return handler(ctx, req)
}
