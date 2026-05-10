package grpc

import (
	"context"
	"time"

	pb "github.com/ricehub-io/api/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ScannerClient interface {
	ScanFile(filePath string) (*pb.ScanResult, error)
}

var Scanner ScannerClient

type fileScanner struct {
	conn   *grpc.ClientConn
	client pb.ScannerClient
}

// InitScanner creates a gRPC connection to the file scanner and sets Scanner global variable.
func InitScanner(url string) {
	logger := zap.L()

	conn, err := grpc.NewClient(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Fatal("Could not connect with file scanner via gRPC",
			zap.Error(err),
			zap.String("url", url),
		)
	}

	Scanner = &fileScanner{conn: conn, client: pb.NewScannerClient(conn)}
	logger.Info("Created gRPC connection with file scanner")
}

// CloseScanner closes the underlying gRPC connection.
func CloseScanner() error {
	if fs, ok := Scanner.(*fileScanner); ok {
		return fs.conn.Close()
	}
	return nil
}

func (s *fileScanner) ScanFile(filePath string) (*pb.ScanResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	return s.client.ScanFile(ctx, &pb.ScanRequest{FilePath: filePath})
}
