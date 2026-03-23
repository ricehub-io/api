package utils

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"
)

func init() {
	// initialize config values for testing
	Config.Blacklist.Usernames = []string{"admin"}
	Config.Blacklist.DisplayNames = []string{"root", "badname"}
	Config.Blacklist.Words = []string{"hate", "spam"}
}

// create a fake stuff to test the 'validateArchive' function
type fakeFile struct{ *bytes.Reader }

func (f *fakeFile) Close() error {
	return nil
}

func (f *fakeFile) Seek(offset int64, whence int) (int64, error) {
	return f.Reader.Seek(offset, whence)
}

type fakeOpener struct {
	content  []byte
	failOpen bool
}

func (fo *fakeOpener) Open() (multipart.File, error) {
	if fo.failOpen {
		return nil, errors.New("disk error")
	}
	return &fakeFile{bytes.NewReader(fo.content)}, nil
}

// buildZipBytes returns a valid in-memory zip archive.
func buildZipBytes(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	f, err := w.Create("hello.txt")
	if err != nil {
		t.Fatalf("zip.Create: %v", err)
	}

	if _, err = io.WriteString(f, "hello"); err != nil {
		t.Fatalf("zip.Write: %v", err)
	}

	if err = w.Close(); err != nil {
		t.Fatalf("zip.Close: %v", err)
	}

	return buf.Bytes()
}

// #################################################
// ############## ValidateFileAsImage ##############
// #################################################
func makeFileHeader(fileName string) *multipart.FileHeader {
	h := textproto.MIMEHeader{
		"tContent-Disposition": []string{
			`form-data; name="file"; filename="` + fileName + `"`,
		},
	}
	return &multipart.FileHeader{
		Filename: fileName,
		Header:   h,
	}
}

func TestValidateFileAsImage_PNG(t *testing.T) {
	ext, err := ValidateFileAsImage(makeFileHeader("pfp.png"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ext != ".png" {
		t.Errorf("expected '.png', got '%s'", ext)
	}
}

func TestValidateFileAsImage_JPG(t *testing.T) {
	ext, err := ValidateFileAsImage(makeFileHeader("pfp.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ext != ".jpg" {
		t.Errorf("expected '.jpg', got '%s'", ext)
	}
}

func TestValidateFileAsImage_UnsupportedExtension(t *testing.T) {
	_, err := ValidateFileAsImage(makeFileHeader("doc.pdf"))
	if err == nil {
		t.Error("expected error for unsupported file type, got nil")
	}
}

func TestValidateFileAsImage_NoExtension(t *testing.T) {
	_, err := ValidateFileAsImage(makeFileHeader("noextdummy"))
	if err == nil {
		t.Error("expected error for file with no extension")
	}
}

func TestValidateFileAsImage_GIF_NotAccepted(t *testing.T) {
	_, err := ValidateFileAsImage(makeFileHeader("anim.gif"))
	if err == nil {
		t.Error("expected error for '.gif'")
	}
}

func TestValidateFileAsImage_CaseInsensitive(t *testing.T) {
	_, err := ValidateFileAsImage(makeFileHeader("prof.PNG"))
	if err != nil {
		t.Error("expected no error for case-insensitive '.PNG'")
	}
}

// #################################################
// ############ ContainsBlacklistedWord ############
// #################################################
func TestContainsBlacklistedWord_MatchesWholeWord(t *testing.T) {
	if !ContainsBlacklistedWord("this is bad text", []string{"bad"}) {
		t.Error("expected 'bad' to be found as a whole word")
	}
}

func TestContainsBlacklistedWord_DoesNotMatchSubstring(t *testing.T) {
	if ContainsBlacklistedWord("classroom", []string{"room"}) {
		t.Error("should not match 'room' inside 'classroom'")
	}
}

func TestContainsBlacklistedWord_CaseInsensitive(t *testing.T) {
	if !ContainsBlacklistedWord("This Is BAD", []string{"bad"}) {
		t.Error("expected case-insensitive match for 'bad'")
	}
}

func TestContainsBlacklistedWord_EmptyText(t *testing.T) {
	if ContainsBlacklistedWord("", []string{"bad"}) {
		t.Error("empty text should not match")
	}
}

func TestContainsBlacklistedWord_MultipleWords_FirstMatches(t *testing.T) {
	if !ContainsBlacklistedWord("spam and bulk", []string{"spam", "eggs"}) {
		t.Error("expected 'spam' to match")
	}
}

func TestContainsBlacklistedWord_MultipleWords_SecondMatches(t *testing.T) {
	if !ContainsBlacklistedWord("experienced and quick", []string{"slow", "experienced"}) {
		t.Error("expected 'experienced' to match")
	}
}

func TestContainsBlacklistedWord_NoMatch(t *testing.T) {
	if ContainsBlacklistedWord("hello, world", []string{"bad", "quick", "wrong"}) {
		t.Error("expected no match")
	}
}

func TestContainsBlacklistedWord_WordAtStart(t *testing.T) {
	if !ContainsBlacklistedWord("bad actor", []string{"bad"}) {
		t.Error("expected match at the start of the string")
	}
}

func TestContainsBlacklistedWord_WordAtEnd(t *testing.T) {
	if !ContainsBlacklistedWord("ultra bad", []string{"bad"}) {
		t.Error("expected match at the end of the string")
	}
}

// #################################################
// ############# IsUsernameBlacklisted #############
// #################################################
func TestIsUsernameBlacklisted_ExactMatch(t *testing.T) {
	if !IsUsernameBlacklisted("admin") {
		t.Error("expected 'admin' to be blacklisted")
	}
}

func TestIsUsernameBlacklisted_ExactMatch_CaseInsensitive(t *testing.T) {
	if !IsUsernameBlacklisted("ADMIN") {
		t.Error("expected case-insensitive exact match for 'ADMIN'")
	}
}

func TestIsUsernameBlacklisted_NotBlacklisted(t *testing.T) {
	if IsUsernameBlacklisted("testuser") {
		t.Error("expected 'testuser' to not be blacklisted")
	}
}

func TestIsUsernameBlacklisted_ContainsBannedWord(t *testing.T) {
	if !IsUsernameBlacklisted("hate") {
		t.Error("expected username containing banned word to be blacklisted")
	}
}

func TestIsUsernameBlacklisted_BannedWordAsSubstring_NoMatch(t *testing.T) {
	if IsUsernameBlacklisted("hater") {
		t.Error("should not match 'hate' as a substring of 'hater'")
	}
}

func TestIsUsernameBlacklisted_DisplayName_ExactMatch(t *testing.T) {
	if !IsUsernameBlacklisted("root") {
		t.Error("expected 'root' (from DisplayNames list) to be blacklisted")
	}
}

// #################################################
// ########### IsDisplayNameBlacklisted ############
// #################################################
func TestIsDisplayNameBlacklisted_ContainsBannedWord(t *testing.T) {
	if !IsDisplayNameBlacklisted("I love spam") {
		t.Error("expected display name containing 'spam' to be blacklisted")
	}
}

func TestIsDisplayNameBlacklisted_ContainsBannedDisplayName(t *testing.T) {
	if !IsDisplayNameBlacklisted("I have a badname") {
		t.Error("expected display name containing banned display name to be blacklisted")
	}
}

func TestIsDisplayNameBlacklisted_NotBlacklisted(t *testing.T) {
	if IsDisplayNameBlacklisted("hello, world") {
		t.Error("expected 'hello, world' to not be blacklisted")
	}
}

func TestIsDisplayNameBlacklisted_CaseInsensitive(t *testing.T) {
	if !IsDisplayNameBlacklisted("SPAM") {
		t.Error("expected case-insensitive match for 'SPAM'")
	}
}

// #################################################
// ################ validateArchive ################
// #################################################

func TestValidateArchive_ValidZip(t *testing.T) {
	o := &fakeOpener{content: buildZipBytes(t)}

	ext, err := validateArchive(o)
	if err != nil {
		t.Fatalf("expected no error for a valid zip, got: %v", err)
	}
	if ext != ".zip" {
		t.Errorf("expected extension '.zip', got '%s'", ext)
	}
}

func TestValidateArchive_OpenFails(t *testing.T) {
	o := &fakeOpener{failOpen: true}

	_, err := validateArchive(o)
	if err == nil {
		t.Fatal("expected error when Open() fails, got nil")
	}
}

func TestValidateArchive_NotAZip_PlainText(t *testing.T) {
	o := &fakeOpener{content: []byte("lorem ipsum plainum textum okok?")}

	_, err := validateArchive(o)
	if err == nil {
		t.Fatal("expected error for plain-text content, got nil")
	}
}

func TestValidateArchive_NotAZip_PDF(t *testing.T) {
	// imitate a pdf file using magic bytes
	pdf := append([]byte("%PDF-1.4\n"), make([]byte, 100)...)
	o := &fakeOpener{content: pdf}

	_, err := validateArchive(o)
	if err == nil {
		t.Fatal("expected error for PDF content disguised as an archive")
	}
}

func TestValidateArchive_NotAZip_PNG(t *testing.T) {
	// use png magic bytes
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	o := &fakeOpener{content: png}

	_, err := validateArchive(o)
	if err == nil {
		t.Fatal("expected error for PNG bytes, only zip is accepted")
	}
}

func TestValidateArchive_EmptyFile(t *testing.T) {
	o := &fakeOpener{content: []byte{}}

	_, err := validateArchive(o)
	if err == nil {
		t.Fatal("expected error for empty file content")
	}
}

func TestValidateArchive_TarGz_NotAccepted(t *testing.T) {
	// tar.gz magic bytes
	tarGz := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00}
	o := &fakeOpener{content: tarGz}

	_, err := validateArchive(o)
	if err == nil {
		t.Fatal("expected error for tar.gz — only zip is accepted")
	}
}

func TestValidateArchive_ZipWithWrongExtension(t *testing.T) {
	// make sure that the validation checks magic bytes
	// even if the provided file extension is not correct
	// (we change it anyway before saving on disk)
	o := &fakeOpener{content: buildZipBytes(t)}

	_, err := validateArchive(o)
	if err != nil {
		t.Fatalf("expected valid zip bytes to pass regardless of filename: %v", err)
	}
}
