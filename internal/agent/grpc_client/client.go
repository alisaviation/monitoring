// Package grpc_client provides a Go client for sending metrics to a gRPC-based monitoring service.
package grpc_client

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/proto/brief/rpc"
)

// GRPCClient provides a client for sending metrics to a gRPC monitoring service.
// It supports encryption, hash verification, and TLS security.
type GRPCClient struct {
	conn          *grpc.ClientConn
	client        rpc.MonitoringServiceClient
	useEncryption bool
	key           string
	publicKey     *rsa.PublicKey
}

// NewGRPCClient creates a new gRPC client connection to the monitoring service.
// Parameters:
//   - address: Server address in format "host:port"
//   - useTLS: Enable TLS encryption for the connection
//   - key: Secret key for request hash calculation (optional)
//   - publicKey: RSA public key for request encryption (optional)
func NewGRPCClient(address string, useTLS bool, key string, publicKey *rsa.PublicKey) (*GRPCClient, error) {
	var opts []grpc.DialOption

	if useTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, err
	}

	return &GRPCClient{
		conn:      conn,
		client:    rpc.NewMonitoringServiceClient(conn),
		key:       key,
		publicKey: publicKey,
	}, nil
}

// SendMetricBatch sends a batch of metrics to the monitoring service.
// Automatically handles encryption and hash verification based on client configuration.
// If encryption is enabled, the entire request payload is encrypted using RSA-AES hybrid encryption.
// If a key is provided, calculates and includes SHA256 hash for request verification.
func (c *GRPCClient) SendMetricBatch(ctx context.Context, metrics map[string]*models.Metric) error {
	protoMetrics := make([]*rpc.Metric, 0, len(metrics))

	for _, metric := range metrics {
		protoMetrics = append(protoMetrics, c.modelToProto(metric))
	}

	req := &rpc.UpdateRequest{
		Metrics: protoMetrics,
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	md := metadata.New(nil)

	if c.useEncryption && c.publicKey != nil {
		reqData, err := proto.Marshal(req)
		if err != nil {
			return err
		}

		encryptedData, err := helpers.EncryptData(reqData, c.publicKey)
		if err != nil {
			return err
		}

		encryptedReq := &rpc.UpdateRequest{
			EncryptedData: encryptedData,
		}

		md.Set("x-content-encrypted", "hybrid-rsa-aes")
		ctx = metadata.NewOutgoingContext(ctx, md)

		_, err = c.client.Update(ctx, encryptedReq)
		return err
	}

	if c.key != "" {
		reqData, err := proto.Marshal(req)
		if err != nil {
			return err
		}
		hash := helpers.CalculateHash(reqData, c.key)
		md.Set("hashsha256", hash)
	}

	ctx = metadata.NewOutgoingContext(ctx, md)
	_, err := c.client.Update(ctx, req)
	return err
}

func (c *GRPCClient) modelToProto(metric *models.Metric) *rpc.Metric {
	protoMetric := &rpc.Metric{
		Id:    metric.ID,
		Mtype: metric.MType,
		Hash:  metric.Hash,
	}

	if metric.MType == models.Counter && metric.Delta != nil {
		protoMetric.Delta = *metric.Delta
	}

	if metric.MType == models.Gauge && metric.Value != nil {
		protoMetric.Value = *metric.Value
	}

	return protoMetric
}

// Close gracefully closes the gRPC connection to the server.
func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

// SetUseEncryption enables or disables request encryption for subsequent requests.
func (c *GRPCClient) SetUseEncryption(use bool) {
	c.useEncryption = use
}
