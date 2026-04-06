package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"ricehub/internal/errs"
	"ricehub/internal/grpc"
	"ricehub/internal/validation"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const dotfilesDir = "dotfiles"

type namedCloser interface {
	io.Closer
	Name() string
}

func HandleDotfilesUpload(fileHeader *multipart.FileHeader) (string, error) {
	logger := zap.L()

	ext, err := validation.ValidateFileAsArchive(fileHeader)
	if err != nil {
		return "", err
	}

	// open file
	file, err := fileHeader.Open()
	if err != nil {
		return "", errs.InternalError(err)
	}
	defer closeSilent(file)

	// create new named temp file
	tmp, err := os.CreateTemp("", "dotfiles-*.zip")
	if err != nil {
		return "", errs.InternalError(err)
	}

	// clean up
	tmpPath := tmp.Name()
	defer func() {
		if err := os.Remove(tmpPath); err != nil {
			logger.Error("Failed to remove temp dotfiles",
				zap.Error(err),
				zap.String("path", tmpPath),
			)
		}
	}()

	// copy dotfiles data into temp file
	if _, err := io.Copy(tmp, file); err != nil {
		closeLog(tmp)
		return "", errs.InternalError(err)
	}
	closeLog(tmp)

	// scan 'em
	res, err := grpc.Scanner.ScanFile(tmpPath)
	if err != nil {
		return "", errs.InternalError(err)
	}
	if res.IsMalicious {
		logger.Warn("Malicious dotfiles detected",
			zap.Strings("findings", res.Reason),
		)
		return "", errs.UserError(
			"Malicious content detected inside dotfiles",
			http.StatusUnprocessableEntity,
		)
	}

	// move to destination path if clean
	tmpRoot, err := os.OpenRoot(os.TempDir())
	if err != nil {
		return "", errs.InternalError(err)
	}
	defer closeLog(tmpRoot)

	destRoot, err := os.OpenRoot(fmt.Sprintf("./public/%s", dotfilesDir))
	if err != nil {
		return "", errs.InternalError(err)
	}
	defer closeLog(destRoot)

	destName := fmt.Sprintf("%v%s", uuid.New(), ext)
	if err := moveFile(tmpRoot, destRoot, filepath.Base(tmpPath), destName); err != nil {
		return "", errs.InternalError(err)
	}

	return fmt.Sprintf("/%s/%s", dotfilesDir, destName), nil
}

func moveFile(srcRoot, destRoot *os.Root, srcName, destName string) error {
	srcPath := filepath.Join(srcRoot.Name(), srcName)
	destPath := filepath.Join(destRoot.Name(), destName)
	if err := os.Rename(srcPath, destPath); err == nil {
		return nil
	}

	// fallback: copy then delete
	in, err := srcRoot.Open(srcName)
	if err != nil {
		return err
	}
	defer closeSilent(in)

	out, err := destRoot.Create(destName)
	if err != nil {
		return err
	}
	defer closeLog(out)

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	if err := out.Sync(); err != nil {
		return err
	}

	return nil
}

// closeLog tries to close given file descriptor.
// Creates new log if close failed. Doesn't panic.
func closeLog(file namedCloser) {
	if err := file.Close(); err != nil {
		zap.L().Error("Failed to close file",
			zap.Error(err),
			zap.String("path", file.Name()),
		)
	}
}

// closeSilent tries to close given file and ignores any errors.
// Useful when deferring close for read-only files.
func closeSilent(file io.Closer) {
	_ = file.Close()
}
