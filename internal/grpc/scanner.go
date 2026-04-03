package grpc

import (
	"context"
	"time"

	pb "ricehub/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Wrapper over gRPC scanner client
type FileScanner struct {
	conn   *grpc.ClientConn
	client pb.ScannerClient
}

var Scanner FileScanner

// Initializes global `scanner` variable and tries to create
// new gRPC connection with file scanner for given URL.
func (s *FileScanner) Init(url string) {
	logger := zap.L()

	var err error
	s.conn, err = grpc.NewClient(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Fatal("Could not connect with file scanner via gRPC",
			zap.Error(err),
			zap.String("url", url),
		)
	}

	s.client = pb.NewScannerClient(s.conn)
	logger.Info("Created gRPC connection with file scanner")
}

// Closes internal gRPC connection.
func (s *FileScanner) Close() {
	s.conn.Close()
}

func (s *FileScanner) ScanFile(filePath string) (res *pb.ScanResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	res, err = s.client.ScanFile(ctx, &pb.ScanRequest{FilePath: filePath})
	return
}
