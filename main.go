package grpcclientlb

import (
	"errors"
	"fmt"
	"sync/atomic"

	"google.golang.org/grpc"
)

type GrpcClientLB interface {
	Get() *grpc.ClientConn
	Close() error
}

type grpcClientLB struct {
	grpcClient []*grpc.ClientConn
	size       uint32
	offset     uint32
}

func NewGrpcClientLB(factory func() (*grpc.ClientConn, error), poolSize int) (GrpcClientLB, error) {
	if poolSize <= 0 {
		return nil, errors.New("poolSize must be greater than 0")
	}

	grpcClient := make([]*grpc.ClientConn, poolSize)
	for i := 0; i < poolSize; i++ {
		conn, err := factory()
		if err != nil {
			for j := 0; j < i; j++ {
				_ = grpcClient[j].Close()
			}
			return nil, fmt.Errorf("failed to create connection at index %d: %w", i, err)
		}

		grpcClient[i] = conn
	}

	return &grpcClientLB{
		grpcClient: grpcClient,
		size:       uint32(poolSize),
		offset:     0,
	}, nil
}

func (g *grpcClientLB) Get() *grpc.ClientConn {
	n := atomic.AddUint32(&g.offset, 1)

	return g.grpcClient[(n-1)%g.size]
}

func (g *grpcClientLB) Close() error {
	var errs []error
	for _, conn := range g.grpcClient {
		if err := conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close %d connections", len(errs))
	}
	return nil
}
