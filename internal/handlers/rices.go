package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"ricehub/internal/grpc"
	"ricehub/internal/models"
	"ricehub/internal/polar"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"ricehub/internal/validation"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

const dotfilesDir = "dotfiles"

type ricesPath struct {
	RiceID string `uri:"id" binding:"required,uuid"`
}

var availableSorts = []string{"trending", "recent", "mostDownloads", "mostStars"}

func checkCanUserModifyRice(token *security.AccessToken, riceID string) error {
	if token.IsAdmin {
		return nil
	}

	isAuthor, err := repository.UserOwnsRice(riceID, token.Subject)
	if err != nil || !isAuthor {
		return errs.NoAccess
	}

	return nil
}

func fetchWaitingRices(c *gin.Context) {
	rices, err := repository.FetchWaitingRices()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.PartialRicesToDTO(rices))
}

func FetchRices(c *gin.Context) {
	token := GetTokenFromRequest(c)
	isAdmin := token != nil && token.IsAdmin

	// TODO: make fields required if others are present (https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Baked_In_Validators_and_Tags)
	var query struct {
		Sort          string     `form:"sort,default=trending"`
		State         string     `form:"state"`
		LastID        *string    `form:"lastId" binding:"omitempty,uuid"`
		LastScore     *float32   `form:"lastScore"`
		LastCreatedAt *time.Time `form:"lastCreatedAt"`
		LastStars     *int       `form:"lastStars"`
		LastDownloads *int       `form:"lastDownloads"`
		Reverse       bool       `form:"reverse"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		// TODO: return different message depending on which parameter was invalid
		c.Error(errs.UserError(
			"Failed to parse query parameters",
			http.StatusBadRequest,
		))
		return
	}

	// check if user is an admin and can filter by state
	if query.State != "" && isAdmin {
		fetchWaitingRices(c)
		return
	}

	if !slices.Contains(availableSorts, query.Sort) {
		c.Error(errs.UserError("Unsupported sorting method provided", http.StatusBadRequest))
		return
	}

	var pag repository.Pagination

	pag.LastID = query.LastID
	pag.LastScore = query.LastScore
	pag.LastCreatedAt = query.LastCreatedAt
	pag.LastDownloads = query.LastDownloads
	pag.LastStars = query.LastStars
	pag.Reverse = query.Reverse

	rices := []models.PartialRice{}
	var err error

	var userID *string = nil
	if token != nil {
		userID = &token.Subject
	}

	switch query.Sort {
	case "trending":
		rices, err = repository.FetchTrendingRices(&pag, userID)
	case "recent":
		rices, err = repository.FetchRecentRices(&pag, userID)
	case "mostDownloads":
		rices, err = repository.FetchMostDownloadedRices(&pag, userID)
	case "mostStars":
		rices, err = repository.FetchMostStarredRices(&pag, userID)
	}

	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	pages, err := repository.FetchPageCount()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	if pag.Reverse {
		// reverse rice array
		for i, j := 0, len(rices)-1; i < j; i, j = i+1, j-1 {
			rices[i], rices[j] = rices[j], rices[i]
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"pageCount": pages,
		"rices":     models.PartialRicesToDTO(rices),
	})
}

func GetRiceByID(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	token := GetTokenFromRequest(c)
	userID := GetUserIDFromRequest(c)

	rice, err := repository.FindRiceByID(userID, path.RiceID)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.RiceNotFound))
		return
	}

	if rice.Rice.State == models.Waiting && (token == nil || !token.IsAdmin) {
		c.Error(errs.RiceNotFound)
		return
	}

	c.JSON(http.StatusOK, rice.ToDTO())
}

func GetRiceComments(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	comments, err := repository.FetchCommentsByRiceID(path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.CommentsWithUserToDTO(comments))
}

func DownloadDotfiles(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	riceID, _ := uuid.Parse(path.RiceID)
	userID := GetUserIDFromRequest(c)

	// try to find the rice
	rice, err := repository.FindRiceByID(userID, path.RiceID)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.RiceNotFound))
		return
	}

	// check if user can download dotfiles
	if rice.Dotfiles.Type != models.Free && !rice.IsOwned {
		c.Error(errs.UserError("You don't have access to these dotfiles", http.StatusForbidden))
		return
	}

	// increment download count
	filePath, err := repository.IncrementDownloadCount(path.RiceID)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.RiceNotFound))
		return
	}

	// insert download event
	if err := repository.InsertRiceDownload(riceID, userID); err != nil {
		zap.L().Error(
			"Failed to insert download event",
			zap.Error(err),
			zap.String("rice_id", path.RiceID),
			zap.String("user_id", userID.String()),
		)
	}

	fullPath := "./public" + filePath

	ext := filepath.Ext(filePath)
	timestamp := time.Now().UTC().Format("20060102-150405")
	fileName := fmt.Sprintf("%s-%s%s", slug.Make(rice.Rice.Title), timestamp, ext)

	c.FileAttachment(fullPath, fileName)
}

type NamedCloser interface {
	io.Closer
	Name() string
}

// closeLog tries to close given file descriptor.
// Creates new log if close failed. Doesn't panic.
func closeLog(file NamedCloser) {
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

// TODO: get that thing out of here right meow
func handleDotfilesUpload(fileHeader *multipart.FileHeader) (string, error) {
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

func CreateRice(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if _, err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	// validate everything first
	form, err := c.MultipartForm()
	if err != nil {
		c.Error(errs.UserError("Invalid multipart form", http.StatusBadRequest))
		return
	}

	var metadata models.CreateRiceDTO
	if err := validation.ValidateForm(c, &metadata); err != nil {
		c.Error(err)
		return
	}

	screenshots := form.File["screenshots[]"]
	formDotfiles := form.File["dotfiles"]

	if len(screenshots) == 0 {
		c.Error(errs.UserError(
			"At least one screenshot is required",
			http.StatusBadRequest,
		))
		return
	}

	maxPreviews := config.Config.Limits.MaxPreviewsPerRice
	if int64(len(screenshots)) > maxPreviews {
		c.Error(errs.UserError(
			fmt.Sprintf(
				"You cannot add more than %v screenshots",
				maxPreviews,
			),
			http.StatusRequestEntityTooLarge,
		))
		return
	}

	if len(formDotfiles) == 0 {
		c.Error(errs.UserError("Dotfiles are required", http.StatusBadRequest))
		return
	}
	dotfilesFile := formDotfiles[0]

	validPreviews := make(map[string]*multipart.FileHeader, len(screenshots))
	for _, preview := range screenshots {
		ext, err := validation.ValidateFileAsImage(preview)
		if err != nil {
			c.Error(err)
			return
		}

		previewPath := fmt.Sprintf("/previews/%v%v", uuid.New(), ext)
		validPreviews[previewPath] = preview
	}

	dotfilesPath, err := handleDotfilesUpload(dotfilesFile)
	if err != nil {
		c.Error(err)
		return
	}

	hasTags := metadata.Tags != "" && metadata.Tags != "[]"

	var tags []int
	if hasTags {
		if err := json.Unmarshal([]byte(metadata.Tags), &tags); err != nil {
			c.Error(errs.UserError("Failed to parse tags", http.StatusBadRequest))
			return
		}
	}

	// check if title or description contains blacklisted words
	bl := config.Config.Blacklist.Words
	if validation.ContainsBlacklistedWord(metadata.Title, bl) {
		c.Error(errs.BlacklistedRiceTitle)
		return
	}
	if validation.ContainsBlacklistedWord(metadata.Description, bl) {
		c.Error(errs.BlacklistedRiceDescription)
		return
	}

	// scan dotfiles for malicious things

	// end validating

	ctx := context.Background()
	tx, err := repository.StartTx(ctx)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	defer tx.Rollback(ctx)

	// insert the rice base (we need rice id for db relation)
	rice, err := repository.InsertRice(tx, token.Subject, metadata.Title, slug.Make(metadata.Title), metadata.Description, token.IsAdmin)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			c.Error(errs.UserError("Provided rice title is already in use!", http.StatusConflict))
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	for path, file := range validPreviews {
		if err := c.SaveUploadedFile(file, "./public"+path); err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		if err := repository.InsertRiceScreenshotTx(tx, rice.ID, path); err != nil {
			c.Error(errs.InternalError(err))
			return
		}
	}

	// create new polar product if dotfiles are paid
	var productID *string

	if metadata.DotfilesType != nil && *metadata.DotfilesType != models.Free {
		res, err := polar.CreateProduct(metadata.Title, *metadata.DotfilesPrice)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		productID = &res.Product.ID
	}

	dotfilesSize := dotfilesFile.Size
	_, err = repository.InsertRiceDotfiles(tx, rice.ID, dotfilesPath, dotfilesSize, metadata.DotfilesType, metadata.DotfilesPrice, productID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// attach tags
	if hasTags {
		if err := repository.InsertRiceTagsTx(tx, rice.ID, tags); err != nil {
			c.Error(errs.InternalError(err))
			return
		}
	}

	// finish the tx
	if err := tx.Commit(ctx); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusCreated)
}

func UpdateRiceMetadata(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if _, err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	var metadata *models.UpdateRiceDTO
	if err := validation.ValidateJSON(c, &metadata); err != nil {
		c.Error(err)
		return
	}

	if metadata.Title == nil && metadata.Description == nil {
		c.Error(errs.UserError("No field to update provided", http.StatusBadRequest))
		return
	}

	// check against blacklisted words
	bl := config.Config.Blacklist.Words
	if metadata.Title != nil && validation.ContainsBlacklistedWord(*metadata.Title, bl) {
		c.Error(errs.BlacklistedRiceTitle)
		return
	}
	if metadata.Description != nil && validation.ContainsBlacklistedWord(*metadata.Description, bl) {
		c.Error(errs.BlacklistedRiceDescription)
		return
	}

	err := repository.UpdateRice(path.RiceID, metadata.Title, metadata.Description)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusCreated)
}

func AttachTags(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if _, err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	var body models.AttachTagsDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	riceID, _ := uuid.Parse(path.RiceID)
	if err := repository.InsertRiceTags(riceID, body.Tags); err != nil {
		c.Error(errs.FromDBError(err, errs.RiceNotFound))
		return
	}
}

func UnattachTags(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if _, err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	var body models.UnattachTagsDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	riceID, _ := uuid.Parse(path.RiceID)
	if err := repository.DeleteRiceTags(riceID, body.Tags); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func UpdateDotfiles(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if _, err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.Error(errs.MissingFile)
		return
	}

	// delete old dotfiles (if exist)
	oldDotfiles, err := repository.FetchRiceDotfilesPath(path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if oldDotfiles != nil {
		path := "./public" + *oldDotfiles
		if err := os.Remove(path); err != nil {
			zap.L().Error("Failed to remove old dotfiles from storage",
				zap.String("path", path),
			)
		}
	}

	filePath, err := handleDotfilesUpload(file)
	if err != nil {
		c.Error(err)
		return
	}

	fileSize := file.Size
	df, err := repository.UpdateRiceDotfiles(path.RiceID, filePath, fileSize)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, df.ToDTO())
}

// TODO: a lot of duplicated code in endpoints, please encapsulate it into separate function that will be called by all handlers
func UpdateDotfilesType(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if _, err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	var update *models.UpdateDotfilesTypeDTO
	if err := validation.ValidateJSON(c, &update); err != nil {
		c.Error(err)
		return
	}

	ctx := context.Background()
	tx, err := repository.StartTx(ctx)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	defer tx.Rollback(ctx)

	var productID *string

	if update.NewType == models.Free {
		// hide existing product
		existingProdID, err := repository.FindDotfilesProductID(tx, path.RiceID)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		temp := existingProdID.String()
		productID = &temp

		_, err = polar.HideProduct(temp)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}
	} else {
		data, err := repository.FindRiceWithDotfilesByID(tx, path.RiceID)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		if data.Dotfiles.ProductID != nil {
			idStr := data.Dotfiles.ProductID.String()

			// product already exists, unhide it
			_, err := polar.ShowProduct(idStr)
			if err != nil {
				c.Error(errs.InternalError(err))
				return
			}

			productID = &idStr
		} else {
			// create new product
			res, err := polar.CreateProduct(data.Rice.Title, data.Dotfiles.Price)
			if err != nil {
				c.Error(errs.InternalError(err))
				return
			}

			productID = &res.Product.ID
		}
	}

	// update dotfiles in db
	updated, err := repository.UpdateDotfilesType(tx, path.RiceID, update.NewType, productID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !updated {
		c.Error(errs.UserError("Failed to update dotfiles type, please try again later.", http.StatusInternalServerError))
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusOK)
}

func UpdateDotfilesPrice(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if _, err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	var update *models.UpdateDotfilesPriceDTO
	if err := validation.ValidateJSON(c, &update); err != nil {
		c.Error(err)
		return
	}

	// create new db tx
	ctx := context.Background()
	tx, err := repository.StartTx(ctx)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	defer tx.Rollback(ctx)

	// try to update dotfiles price in db
	productID, err := repository.UpdateDotfilesPrice(tx, path.RiceID, update.NewPrice)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// try update product price in polar
	_, err = polar.UpdatePrice(productID.String(), update.NewPrice)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// finish the tx
	if err := tx.Commit(ctx); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusOK)
}

func AddScreenshot(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if _, err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.Error(errs.UserError("Invalid multipart form", http.StatusBadRequest))
		return
	}

	files := form.File["files[]"]
	if len(files) == 0 {
		c.Error(errs.MissingFile)
		return
	}

	count, err := repository.FetchRiceScreenshotCount(path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	maxPreviews := config.Config.Limits.MaxPreviewsPerRice
	if int64(count+len(files)) > maxPreviews {
		c.Error(errs.UserError(
			fmt.Sprintf(
				"You can't have more than %v previews per rice!",
				maxPreviews,
			),
			http.StatusRequestEntityTooLarge,
		))
		return
	}

	type validFile struct {
		path   string
		header *multipart.FileHeader
	}

	validFiles := make([]validFile, 0, len(files))
	for _, file := range files {
		ext, err := validation.ValidateFileAsImage(file)
		if err != nil {
			c.Error(err)
			return
		}
		validFiles = append(validFiles, validFile{
			path:   fmt.Sprintf("/previews/%v%v", uuid.New(), ext),
			header: file,
		})
	}

	ctx := context.Background()
	tx, err := repository.StartTx(ctx)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	defer tx.Rollback(ctx)

	previews := make([]string, 0, len(validFiles))
	for _, vf := range validFiles {
		err = c.SaveUploadedFile(vf.header, "./public"+vf.path)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		err := repository.InsertRiceScreenshotTx(tx, path.RiceID, vf.path)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		previews = append(previews, config.Config.App.CDNUrl+vf.path)
	}

	if err := tx.Commit(ctx); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"previews": previews})
}

func UpdateRiceState(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	var update *models.UpdateRiceStateDTO
	if err := validation.ValidateJSON(c, &update); err != nil {
		c.Error(err)
		return
	}

	rice, err := repository.FindRiceByID(nil, path.RiceID)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.RiceNotFound))
		return
	}
	if rice.Rice.State == models.Accepted {
		c.Error(errs.UserError("This rice has been already accepted", http.StatusConflict))
		return
	}

	switch update.NewState {
	case "accepted":
		err := repository.UpdateRiceState(path.RiceID, models.Accepted)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}
		c.Status(http.StatusOK)
	case "rejected":
		_, err := repository.DeleteRice(path.RiceID)
		if err != nil {
			zap.L().Error(
				"Database error when trying to delete rejected rice",
				zap.String("rice_id", path.RiceID),
				zap.Error(err),
			)
			c.Error(errs.InternalError(err))
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func DeleteScreenshot(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if _, err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path struct {
		RiceID    string `uri:"id" binding:"required,uuid"`
		PreviewID string `uri:"previewId" binding:"required,uuid"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "RiceID") {
			msg = errs.InvalidRiceID.Error()
		} else if strings.Contains(msg, "PreviewID") {
			msg = "Invalid preview ID path parameter. It must be a valid UUID."
		}

		c.Error(errs.UserError(msg, http.StatusBadRequest))
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	// check if there's at least one preview before deleting
	count, err := repository.FetchRiceScreenshotCount(path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if count <= 1 {
		c.Error(errs.UserError("You cannot delete this preview! At least one preview is required for a rice.", http.StatusUnprocessableEntity))
		return
	}

	deleted, err := repository.DeleteRiceScreenshot(path.RiceID, path.PreviewID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !deleted {
		c.Error(errs.UserError("Rice preview with provided ID not found", http.StatusNotFound))
		return
	}

	c.Status(http.StatusNoContent)
}

func AddRiceStar(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := repository.InsertRiceStar(path.RiceID, token.Subject); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				c.Status(http.StatusCreated)
				return
			case pgerrcode.ForeignKeyViolation:
				c.Error(errs.RiceNotFound)
				return
			}
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusCreated)
}

func DeleteRiceStar(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := repository.DeleteRiceStar(path.RiceID, token.Subject); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func PurchaseDotfiles(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	// check if rice exists
	rice, err := repository.FindRiceByID(&userID, path.RiceID)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.RiceNotFound))
		return
	}

	// check if dotfiles are paid
	if rice.Dotfiles.Type == models.Free {
		c.Error(errs.UserError(
			"You can't purchase free dotfiles",
			http.StatusBadRequest,
		))
		return
	}

	// check if user owns the dotfiles
	if rice.IsOwned {
		c.Error(errs.UserError(
			"You already own these dotfiles",
			http.StatusConflict,
		))
		return
	}

	// create new checkout session
	res, err := polar.CreateCheckoutSession(token.Subject, *rice.Dotfiles.ProductID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"checkoutUrl": res.Checkout.URL})
}

func DeleteRice(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if _, err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	// create new db transaction
	ctx := context.Background()
	tx, err := repository.StartTx(ctx)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	defer tx.Rollback(ctx)

	// fetch product id before deleting
	productID, err := repository.FindDotfilesProductID(tx, path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// try to delete the rice from database
	deleted, err := repository.DeleteRiceTx(tx, path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !deleted {
		c.Error(errs.RiceNotFound)
		return
	}

	if productID != nil {
		// try to archive the product
		_, err := polar.ArchiveProduct(productID.String())
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}
	}

	// commit transaction
	if err := tx.Commit(ctx); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}
