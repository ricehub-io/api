package utils

import (
	"mime/multipart"
	"net/textproto"
	"testing"
)

func createTestConfig() {
	Config.Blacklist.Usernames = []string{"admin"}
	Config.Blacklist.DisplayNames = []string{"root", "badname"}
	Config.Blacklist.Words = []string{"hate", "spam"}
}

func init() {
	createTestConfig()
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
