package models

import (
	"ricehub/src/utils"
	"testing"
	"time"

	"github.com/google/uuid"
)

func init() {
	utils.Config.App.CDNUrl = "https://cdn.example.com"
	utils.Config.App.DefaultAvatar = "/default.png"
}

// #################################################
// ################## User.ToDTO ###################
// #################################################
func TestUser_ToDTO_FieldMapping(t *testing.T) {
	now := time.Now()
	u := User{
		ID:          uuid.New(),
		Username:    "testuser",
		DisplayName: "Test User",
		IsAdmin:     true,
		IsBanned:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	dto := u.ToDTO()

	if dto.ID != u.ID {
		t.Errorf("ID mismatch")
	}
	if dto.Username != u.Username {
		t.Errorf("Username mismatch")
	}
	if dto.IsAdmin != true {
		t.Error("expected IsAdmin true")
	}
}

func TestUser_ToDTO_TimesAreUTC(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Now().In(loc)
	u := User{CreatedAt: now, UpdatedAt: now}

	dto := u.ToDTO()

	if dto.CreatedAt.Location() != time.UTC {
		t.Error("CreatedAt should be UTC")
	}
	if dto.UpdatedAt.Location() != time.UTC {
		t.Error("UpdatedAt should be UTC")
	}
}

func TestUser_ToDTO_NilAvatar_UsesDefault(t *testing.T) {
	u := User{AvatarPath: nil}
	dto := u.ToDTO()

	want := "https://cdn.example.com/default.png"
	if dto.AvatarUrl != want {
		t.Errorf("want %q, got %q", want, dto.AvatarUrl)
	}
}

func TestUser_ToDTO_WithAvatar_UsesCDNPath(t *testing.T) {
	path := "/avatars/user-123.png"
	u := User{AvatarPath: &path}
	dto := u.ToDTO()

	want := "https://cdn.example.com/avatars/user-123.png"
	if dto.AvatarUrl != want {
		t.Errorf("want %q, got %q", want, dto.AvatarUrl)
	}
}

// #################################################
// ############## RiceDotfiles.ToDTO ###############
// #################################################
func TestRiceDotfiles_ToDTO_FreeType_NilPrice(t *testing.T) {
	df := RiceDotfiles{
		FilePath: "/files/rice.zip",
		Type:     Free,
		Price:    0,
	}

	dto := df.ToDTO()

	if dto.Price != nil {
		t.Errorf("expected nil `Price` for free dotfiles, got %v", dto.Price)
	}
}

func TestRiceDotfiles_ToDTO_OneTimeType_PriceSet(t *testing.T) {
	df := RiceDotfiles{
		FilePath: "/files/rice.zip",
		Type:     OneTime,
		Price:    9.99,
	}

	dto := df.ToDTO()

	if dto.Price == nil {
		t.Fatal("expected `Price` to be set for one-time dotfiles")
	}
	if *dto.Price != 9.99 {
		t.Errorf("want price 9.99, got %v", *dto.Price)
	}
}

func TestRiceDotfiles_ToDTO_FilePathPrefixedWithCDN(t *testing.T) {
	df := RiceDotfiles{FilePath: "/files/rice.zip", Type: Free}
	dto := df.ToDTO()

	want := "https://cdn.example.com/files/rice.zip"
	if dto.FilePath != want {
		t.Errorf("want %q, got %q", want, dto.FilePath)
	}
}

func TestRiceDotfiles_ToDTO_TimesAreUTC(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(loc)
	df := RiceDotfiles{
		Type:      Free,
		CreatedAt: now,
		UpdatedAt: now,
	}

	dto := df.ToDTO()

	if dto.CreatedAt.Location() != time.UTC {
		t.Error("CreatedAt should be UTC")
	}
	if dto.UpdatedAt.Location() != time.UTC {
		t.Error("UpdatedAt should be UTC")
	}
}

// #################################################
// ############### PartialRice.ToDTO ###############
// #################################################
func TestPartialRice_ToDTO_FreeType_IsFreeTrue(t *testing.T) {
	r := PartialRice{DotfilesType: Free}
	if !r.ToDTO().IsFree {
		t.Error("expected IsFree=true for free dotfiles type")
	}
}

func TestPartialRice_ToDTO_OneTimeType_IsFreeFalse(t *testing.T) {
	r := PartialRice{DotfilesType: OneTime}
	if r.ToDTO().IsFree {
		t.Error("expected IsFree=false for one-time dotfiles type")
	}
}

func TestPartialRice_ToDTO_ThumbnailPrefixedWithCDN(t *testing.T) {
	r := PartialRice{Thumbnail: "/thumbs/rice.jpg"}

	want := "https://cdn.example.com/thumbs/rice.jpg"
	if got := r.ToDTO().Thumbnail; got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestPartialRice_ToDTO_CountFieldsMapped(t *testing.T) {
	r := PartialRice{
		StarCount:     10,
		CommentCount:  5,
		DownloadCount: 42,
	}
	dto := r.ToDTO()

	if dto.Stars != 10 || dto.Comments != 5 || dto.Downloads != 42 {
		t.Errorf(
			"count fields not mapped correctly: stars=%d comments=%d downloads=%d",
			dto.Stars, dto.Comments, dto.Downloads,
		)
	}
}

// #################################################
// ################# UserBan.ToDTO #################
// #################################################
func TestUserBan_ToDTO_NilPointers_RemainNil(t *testing.T) {
	b := UserBan{
		ID:        uuid.New(),
		BannedAt:  time.Now(),
		ExpiresAt: nil,
		RevokedAt: nil,
	}
	dto := b.ToDTO()

	if dto.ExpiresAt != nil {
		t.Error("expected ExpiresAt to be nil")
	}
	if dto.RevokedAt != nil {
		t.Error("expected RevokedAt to be nil")
	}
}

func TestUserBan_ToDTO_ExpiresAtIsUTC(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Warsaw")
	exp := time.Now().In(loc)

	b := UserBan{BannedAt: time.Now(), ExpiresAt: &exp}
	dto := b.ToDTO()

	if dto.ExpiresAt.Location() != time.UTC {
		t.Error("ExpiresAt should be converted to UTC")
	}
}

func TestUserBan_ToDTO_RevokedAtIsUTC(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Warsaw")
	rev := time.Now().In(loc)

	b := UserBan{BannedAt: time.Now(), RevokedAt: &rev}
	dto := b.ToDTO()

	if dto.RevokedAt.Location() != time.UTC {
		t.Error("RevokedAt should be converted to UTC")
	}
}

func TestUserBan_ToDTO_BannedAtIsUTC(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Warsaw")

	b := UserBan{BannedAt: time.Now().In(loc)}
	dto := b.ToDTO()

	if dto.BannedAt.Location() != time.UTC {
		t.Error("BannedAt should be UTC")
	}
}

// #################################################
// ############# LeaderboardRice.ToDTO #############
// #################################################
func TestLeaderboardRice_ToDTO_FreeType_IsFreeTrue(t *testing.T) {
	r := LeaderboardRice{DotfilesType: Free}
	if !r.ToDTO().IsFree {
		t.Error("expected IsFree=true for free dotfiles type")
	}
}

func TestLeaderboardRice_ToDTO_OneTimeType_IsFreeFalse(t *testing.T) {
	r := LeaderboardRice{DotfilesType: OneTime}
	if r.ToDTO().IsFree {
		t.Error("expected IsFree=false for one-time dotfiles type")
	}
}

func TestLeaderboardRice_ToDTO_ThumbnailPrefixedWithCDN(t *testing.T) {
	r := LeaderboardRice{Thumbnail: "/thumbs/rice.jpg"}

	want := "https://cdn.example.com/thumbs/rice.jpg"
	if got := r.ToDTO().Thumbnail; got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestLeaderboardRice_ToDTO_CountFieldsMapped(t *testing.T) {
	r := LeaderboardRice{
		StarCount:     10,
		CommentCount:  5,
		DownloadCount: 42,
	}
	dto := r.ToDTO()

	if dto.Stars != 10 || dto.Comments != 5 || dto.Downloads != 42 {
		t.Errorf(
			"count fields not mapped correctly: stars=%d comments=%d downloads=%d",
			dto.Stars, dto.Comments, dto.Downloads,
		)
	}
}

// #################################################
// ############# bulk ToDTO functions ##############
// #################################################
func TestUsersToDTO_PreservesLength(t *testing.T) {
	users := []User{{}, {}, {}}
	if got := len(UsersToDTO(users)); got != 3 {
		t.Errorf("want 3 DTOs, got %d", got)
	}
}

func TestUsersToDTO_EmptySlice(t *testing.T) {
	if got := len(UsersToDTO([]User{})); got != 0 {
		t.Errorf("want 0 DTOs, got %d", got)
	}
}

func TestPartialRicesToDTO_PreservesLength(t *testing.T) {
	rices := []PartialRice{{}, {}}
	if got := len(PartialRicesToDTO(rices)); got != 2 {
		t.Errorf("want 2 DTOs, got %d", got)
	}
}

func TestLeaderboardRices_ToDTO_PreservesLength(t *testing.T) {
	rices := LeaderboardRices{{}, {}, {}, {}}
	if got := len(rices.ToDTO()); got != 4 {
		t.Errorf("want 4 DTOs, got %d", got)
	}
}
