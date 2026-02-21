package ml

import (
	"context"
	"fmt"
	"time"

	pb "github.com/nonobeam/golang-stock-trading/proto/ml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MLClient wraps the gRPC client
type MLClient struct {
	conn   *grpc.ClientConn
	client pb.MLPredictionServiceClient
}

// NewMLClient creates a new client
func NewMLClient(address string) (*MLClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("did not connect: %v", err)
	}
	
	client := pb.NewMLPredictionServiceClient(conn)
	
	return &MLClient{
		conn:   conn,
		client: client,
	}, nil
}

// Close closes the connection
func (c *MLClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// Ping sends a health check request
func (c *MLClient) Ping(ctx context.Context) (*pb.PingResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.client.Ping(ctx, &pb.PingRequest{Message: "ping"})
}

// TriggerTraining triggers model training for a single ticker
func (c *MLClient) TriggerTraining(ctx context.Context, req *pb.TriggerTrainingRequest) (*pb.TriggerTrainingResponse, error) {
	return c.client.TriggerTraining(ctx, req)
}

// RunWeeklyPortfolio runs the weekly portfolio selection pipeline
func (c *MLClient) RunWeeklyPortfolio(ctx context.Context, req *pb.RunWeeklyPortfolioRequest) (*pb.RunWeeklyPortfolioResponse, error) {
	return c.client.RunWeeklyPortfolio(ctx, req)
}

// TriggerBulkRetrain returns the server-side stream for a bulk retrain job
func (c *MLClient) TriggerBulkRetrain(ctx context.Context, req *pb.TriggerBulkRetrainRequest) (pb.MLPredictionService_TriggerBulkRetrainClient, error) {
	return c.client.TriggerBulkRetrain(ctx, req)
}
