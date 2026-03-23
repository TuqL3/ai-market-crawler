package grpcclient

import (
	"context"
	"fmt"
	"log"

	pb "github.com/lukas/ai-aggregator/go-service/gen/aggregator/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn     *grpc.ClientConn
	Analysis pb.AnalysisServiceClient
	Chat     pb.ChatServiceClient
}

func New(ctx context.Context, addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to python service: %w", err)
	}

	log.Printf("Connected to Python gRPC service at %s", addr)

	return &Client{
		conn:     conn,
		Analysis: pb.NewAnalysisServiceClient(conn),
		Chat:     pb.NewChatServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
