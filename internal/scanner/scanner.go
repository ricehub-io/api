package scanner

import (
	"context"
	"fmt"
	"time"

	scannerv1 "github.com/ricehub-io/proto/gen/go/scanner/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ScannerClient interface {
	ScanFile(filePath string) (*scannerv1.ScanFileResponse, error)
}

// TODO: wrap it into a struct - make it a dependency
var Scanner ScannerClient

type fileScanner struct {
	conn   *grpc.ClientConn
	client scannerv1.ScannerServiceClient
}

// InitScanner creates a gRPC connection to the file scanner and sets Scanner global variable.
func InitScanner(connUrl string) error {
	conn, err := grpc.NewClient(
		connUrl,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("grpc new client: %w", err)
	}

	Scanner = &fileScanner{
		conn:   conn,
		client: scannerv1.NewScannerServiceClient(conn),
	}

	return nil
}

// CloseScanner closes the underlying gRPC connection.
func CloseScanner() error {
	if fs, ok := Scanner.(*fileScanner); ok {
		return fs.conn.Close()
	}
	return nil
}

func (s *fileScanner) ScanFile(filePath string) (*scannerv1.ScanFileResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	return s.client.ScanFile(ctx, &scannerv1.ScanFileRequest{FilePath: filePath})
}
