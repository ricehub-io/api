package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// fakeCloser implements namedCloser for closeLog tests.
type fakeCloser struct {
	name      string
	failClose bool
}

func (f *fakeCloser) Close() error {
	if f.failClose {
		return errors.New("close failed")
	}
	return nil
}

func (f *fakeCloser) Name() string {
	return f.name
}

// fakeReadCloser implements io.Closer for closeSilent tests.
type fakeReadCloser struct {
	failClose bool
}

func (f *fakeReadCloser) Close() error {
	if f.failClose {
		return errors.New("close failed")
	}
	return nil
}

// #################################################
// ################# closeSilent ###################
// #################################################
func TestCloseSilent_SuccessfulClose(t *testing.T) {
	closeSilent(&fakeReadCloser{}) // must not panic
}

func TestCloseSilent_ErrorIgnored(t *testing.T) {
	closeSilent(&fakeReadCloser{failClose: true}) // must not panic
}

// #################################################
// ################### closeLog ####################
// #################################################
func TestCloseLog_SuccessfulClose(t *testing.T) {
	closeLog(&fakeCloser{name: "testfile"}) // must not panic
}

func TestCloseLog_ErrorLogged(t *testing.T) {
	// must not panic even when Close() fails (it logs instead)
	closeLog(&fakeCloser{name: "testfile", failClose: true})
}

// #################################################
// ################### moveFile ####################
// #################################################
func openTempRoot(t *testing.T) (string, *os.Root) {
	t.Helper()
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatalf("os.OpenRoot(%s): %v", dir, err)
	}
	t.Cleanup(func() { root.Close() })
	return dir, root
}

func writeToRoot(t *testing.T, root *os.Root, name, content string) {
	t.Helper()
	f, err := root.Create(name)
	if err != nil {
		t.Fatalf("root.Create(%s): %v", name, err)
	}
	if _, err := io.WriteString(f, content); err != nil {
		f.Close()
		t.Fatalf("write %s: %v", name, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", name, err)
	}
}

func readFromDir(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", name, err)
	}
	return string(data)
}

func TestMoveFile_Success(t *testing.T) {
	srcDir, srcRoot := openTempRoot(t)
	destDir, destRoot := openTempRoot(t)

	writeToRoot(t, srcRoot, "src.zip", "zip contents")

	if err := moveFile(srcRoot, destRoot, "src.zip", "dest.zip"); err != nil {
		t.Fatalf("moveFile: %v", err)
	}

	if _, err := os.Stat(filepath.Join(srcDir, "src.zip")); !os.IsNotExist(err) {
		t.Error("expected source file to be gone after move")
	}

	if got := readFromDir(t, destDir, "dest.zip"); got != "zip contents" {
		t.Errorf("content mismatch: want %q, got %q", "zip contents", got)
	}
}

func TestMoveFile_PreservesContent(t *testing.T) {
	want := "dotfile data\nwith newlines\n\x00binary\xff"

	_, srcRoot := openTempRoot(t)
	destDir, destRoot := openTempRoot(t)

	writeToRoot(t, srcRoot, "data.zip", want)

	if err := moveFile(srcRoot, destRoot, "data.zip", "out.zip"); err != nil {
		t.Fatalf("moveFile: %v", err)
	}

	if got := readFromDir(t, destDir, "out.zip"); got != want {
		t.Errorf("content mismatch: want %q, got %q", want, got)
	}
}

func TestMoveFile_DestNameDiffersFromSrc(t *testing.T) {
	_, srcRoot := openTempRoot(t)
	destDir, destRoot := openTempRoot(t)

	writeToRoot(t, srcRoot, "original.zip", "data")

	if err := moveFile(srcRoot, destRoot, "original.zip", "renamed.zip"); err != nil {
		t.Fatalf("moveFile: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "renamed.zip")); err != nil {
		t.Errorf("expected renamed.zip to exist: %v", err)
	}
}

func TestMoveFile_EmptyFile(t *testing.T) {
	_, srcRoot := openTempRoot(t)
	destDir, destRoot := openTempRoot(t)

	writeToRoot(t, srcRoot, "empty.zip", "")

	if err := moveFile(srcRoot, destRoot, "empty.zip", "out.zip"); err != nil {
		t.Fatalf("moveFile: %v", err)
	}

	if got := readFromDir(t, destDir, "out.zip"); got != "" {
		t.Errorf("expected empty content, got %q", got)
	}
}

func TestMoveFile_SourceNotFound(t *testing.T) {
	_, srcRoot := openTempRoot(t)
	_, destRoot := openTempRoot(t)

	err := moveFile(srcRoot, destRoot, "non-existent.zip", "dest.zip")
	if err == nil {
		t.Error("expected error for missing source file, got nil")
	}
}

func TestMoveFile_SameRootDir(t *testing.T) {
	dir, root := openTempRoot(t)

	// second root handle in the same dir
	root2, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	t.Cleanup(func() { root2.Close() })

	writeToRoot(t, root, "a.zip", "payload")

	if err := moveFile(root, root2, "a.zip", "b.zip"); err != nil {
		t.Fatalf("moveFile within same dir: %v", err)
	}

	if got := readFromDir(t, dir, "b.zip"); got != "payload" {
		t.Errorf("content mismatch: got %q", got)
	}
}
