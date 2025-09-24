// Package middleware provides gRPC interceptors for authentication, encryption,
// logging, and other cross-cutting concerns.
package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/storage"
	"github.com/alisaviation/monitoring/proto/brief/rpc"
)

// GRPCInterceptor combines all gRPC middleware components including
// authentication, encryption, logging, and storage synchronization.
type GRPCInterceptor struct {
	privateKey    *rsa.PrivateKey
	key           string
	trustedSubnet string
	storeInterval time.Duration
	storage       storage.Storage
}

// NewGRPCInterceptor creates a new GRPCInterceptor with the provided dependencies.
// Parameters:
//   - privateKey: RSA private key for request decryption (optional)
//   - key: Secret key for hash verification (optional)
//   - trustedSubnet: CIDR notation for allowed IP range (optional)
//   - storeInterval: Metrics storage interval (0 for synchronous storage)
//   - storage: Storage implementation for metrics persistence
func NewGRPCInterceptor(privateKey *rsa.PrivateKey, key, trustedSubnet string, storeInterval time.Duration, storage storage.Storage) *GRPCInterceptor {
	return &GRPCInterceptor{
		privateKey:    privateKey,
		key:           key,
		trustedSubnet: trustedSubnet,
		storeInterval: storeInterval,
		storage:       storage,
	}
}

// UnaryServerInterceptor returns a unary server interceptor that applies
// all middleware components in the following order:
// 1. Request logging
// 2. Trusted subnet validation
// 3. Key context injection
// 4. Request decryption (if encrypted)
// 5. Hash verification (if key provided)
// 6. Synchronous storage backup (for write methods)
// 7. Request handling
// 8. Synchronous storage execution (for successful write operations)
func (gi *GRPCInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = gi.loggingMiddleware(ctx, info)

		ctx, err := gi.trustedSubnetMiddleware(ctx)
		if err != nil {
			return nil, err
		}

		ctx = gi.keyContextMiddleware(ctx)

		ctx, req, err = gi.decryptMiddleware(ctx, req, info)
		if err != nil {
			return nil, err
		}

		ctx, err = gi.hashCheckMiddleware(ctx, req, info)
		if err != nil {
			return nil, err
		}

		var prevGauges map[string]float64
		var prevCounters map[string]int64
		if gi.storeInterval == 0 && gi.isWriteMethod(info.FullMethod) {
			prevGauges, prevCounters = gi.syncSaveBackupMiddleware(ctx)
		}

		resp, err := handler(ctx, req)

		if gi.storeInterval == 0 && gi.isWriteMethod(info.FullMethod) && err == nil {
			gi.syncSaveExecuteMiddleware(ctx, prevGauges, prevCounters)
		}

		gi.logResponse(ctx, info, resp, err)

		return resp, err
	}
}

// StreamServerInterceptor returns a stream server interceptor that applies
// basic middleware components to stream requests (logging, subnet validation, key injection).
// Encryption and hash verification are not supported for stream methods.
func (gi *GRPCInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		ctx = gi.loggingMiddleware(ctx, &grpc.UnaryServerInfo{FullMethod: info.FullMethod})

		var err error
		ctx, err = gi.trustedSubnetMiddleware(ctx)
		if err != nil {
			return err
		}

		ctx = gi.keyContextMiddleware(ctx)

		wrappedStream := &wrappedServerStream{ss, ctx}
		return handler(srv, wrappedStream)
	}
}

func (gi *GRPCInterceptor) loggingMiddleware(ctx context.Context, info *grpc.UnaryServerInfo) context.Context {
	var clientIP string
	if p, ok := peer.FromContext(ctx); ok {
		clientIP = ExtractIP(p.Addr.String())
	}

	logger.Log.Info("gRPC request started",
		zap.String("method", info.FullMethod),
		zap.String("client_ip", clientIP),
	)

	return ctx
}

func (gi *GRPCInterceptor) logResponse(ctx context.Context, info *grpc.UnaryServerInfo, resp interface{}, err error) {
	var clientIP string
	if p, ok := peer.FromContext(ctx); ok {
		clientIP = ExtractIP(p.Addr.String())
	}

	fields := []zap.Field{
		zap.String("method", info.FullMethod),
		zap.String("client_ip", clientIP),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.Log.Error("gRPC request failed", fields...)
	} else {
		logger.Log.Info("gRPC request completed", fields...)
	}
}

func (gi *GRPCInterceptor) trustedSubnetMiddleware(ctx context.Context) (context.Context, error) {
	if gi.trustedSubnet == "" {
		return ctx, nil
	}

	p, ok := peer.FromContext(ctx)
	if !ok {
		logger.Log.Warn("gRPC client information not available")
		return ctx, status.Error(codes.Unauthenticated, "client information not available")
	}

	clientIP := ExtractIP(p.Addr.String())
	if !isIPInSubnetGRPC(clientIP, gi.trustedSubnet) {
		logger.Log.Warn("gRPC access denied from untrusted subnet")
		return ctx, status.Error(codes.PermissionDenied, "access denied")
	}

	logger.Log.Debug("gRPC IP address allowed")
	return ctx, nil
}

func (gi *GRPCInterceptor) keyContextMiddleware(ctx context.Context) context.Context {
	if gi.key != "" {
		return context.WithValue(ctx, secretKey, gi.key)
	}
	return ctx
}

func (gi *GRPCInterceptor) hashCheckMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (context.Context, error) {
	if gi.key == "" {
		return ctx, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, nil
	}

	hashHeaders := md.Get("hashsha256")
	if len(hashHeaders) == 0 {
		return ctx, nil
	}

	clientHash := hashHeaders[0]
	if clientHash == "" {
		return ctx, nil
	}

	logger.Log.Debug("gRPC hash check started")

	reqBytes, err := gi.serializeForHashCheck(req)
	if err != nil {
		logger.Log.Error("Failed to serialize message for hash check",
			zap.String("method", info.FullMethod),
			zap.Error(err))
		return ctx, status.Error(codes.InvalidArgument, "invalid request data")
	}

	computedHash := helpers.CalculateHash(reqBytes, gi.key)
	if computedHash != clientHash {
		logger.Log.Warn("gRPC hash check failed")
		return ctx, status.Error(codes.InvalidArgument, "invalid hash")
	}

	logger.Log.Debug("gRPC hash check successful")
	return ctx, nil
}

func (gi *GRPCInterceptor) decryptMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (context.Context, interface{}, error) {
	if gi.privateKey == nil {
		return ctx, req, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, req, nil
	}

	encryptedFlags := md.Get("x-content-encrypted")
	if len(encryptedFlags) == 0 || encryptedFlags[0] != "hybrid-rsa-aes" {
		return ctx, req, nil
	}

	logger.Log.Debug("gRPC decryption started",
		zap.String("method", info.FullMethod))

	decryptedReq, err := gi.decryptRequest(req, info.FullMethod)
	if err != nil {
		logger.Log.Error("gRPC decryption failed",
			zap.String("method", info.FullMethod),
			zap.Error(err))
		return ctx, req, status.Error(codes.InvalidArgument, "decryption failed")
	}

	logger.Log.Debug("gRPC decryption successful",
		zap.String("method", info.FullMethod))

	return ctx, decryptedReq, nil
}

func (gi *GRPCInterceptor) decryptRequest(req interface{}, method string) (interface{}, error) {
	switch r := req.(type) {
	case *rpc.UpdateRequest:
		if len(r.EncryptedData) > 0 {
			decryptedData, err := helpers.DecryptData(r.EncryptedData, gi.privateKey)
			if err != nil {
				return nil, err
			}

			var decryptedReq rpc.UpdateRequest
			if err := proto.Unmarshal(decryptedData, &decryptedReq); err != nil {
				return nil, err
			}
			return &decryptedReq, nil
		}
		if len(r.Metrics) > 0 {
			return r, nil
		}
	case *rpc.ValueRequest:
		if len(r.EncryptedData) > 0 {
			decryptedData, err := helpers.DecryptData(r.EncryptedData, gi.privateKey)
			if err != nil {
				return nil, err
			}

			var decryptedReq rpc.ValueRequest
			if err := proto.Unmarshal(decryptedData, &decryptedReq); err != nil {
				return nil, err
			}
			return &decryptedReq, nil
		}
		return r, nil
	}

	return req, nil
}

func (gi *GRPCInterceptor) syncSaveBackupMiddleware(ctx context.Context) (map[string]float64, map[string]int64) {
	prevGauges := make(map[string]float64)
	prevCounters := make(map[string]int64)

	if gi.storeInterval == 0 {
		gauges, err := gi.storage.Gauges(ctx)
		if err != nil {
			logger.Log.Error("Error getting gauges for backup", zap.Error(err))
		} else {
			for k, v := range gauges {
				prevGauges[k] = v
			}
		}

		counters, err := gi.storage.Counters(ctx)
		if err != nil {
			logger.Log.Error("Error getting counters for backup", zap.Error(err))
		} else {
			for k, v := range counters {
				prevCounters[k] = v
			}
		}
	}

	return prevGauges, prevCounters
}

func (gi *GRPCInterceptor) syncSaveExecuteMiddleware(ctx context.Context, prevGauges map[string]float64, prevCounters map[string]int64) {
	if gi.storeInterval == 0 {
		helpers.CheckAndSaveMetrics(ctx, gi.storage, prevGauges, prevCounters)
	}
}

func (gi *GRPCInterceptor) isWriteMethod(method string) bool {
	writeMethods := []string{"Update", "Updates"}
	for _, m := range writeMethods {
		if strings.Contains(method, m) {
			return true
		}
	}
	return false
}

func (gi *GRPCInterceptor) serializeForHashCheck(req interface{}) ([]byte, error) {
	if msg, ok := req.(proto.Message); ok {
		return proto.Marshal(msg)
	}
	return json.Marshal(req)
}

// ExtractIP extracts the IP address from a network address string.
// Handles both IP:port format and plain IP addresses.
func ExtractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func isIPInSubnetGRPC(ipStr, subnetStr string) bool {
	ipStr = strings.TrimSpace(ipStr)
	subnetStr = strings.TrimSpace(subnetStr)

	ip := net.ParseIP(ipStr)
	if ip == nil {
		logger.Log.Error("Invalid IP address", zap.String("ip", ipStr))
		return false
	}

	_, subnet, err := net.ParseCIDR(subnetStr)
	if err != nil {
		logger.Log.Error("Invalid subnet CIDR",
			zap.String("subnet", subnetStr),
			zap.Error(err))
		return false
	}

	return subnet.Contains(ip)
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
