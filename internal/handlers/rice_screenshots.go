package handlers

import (
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/security"
	"ricehub/internal/services"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RiceScreenshotHandler struct {
	svc *services.RiceScreenshotService
}

func NewRiceScreenshotHandler(svc *services.RiceScreenshotService) *RiceScreenshotHandler {
	return &RiceScreenshotHandler{svc}
}

func (h *RiceScreenshotHandler) CreateScreenshot(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	form, formErr := c.MultipartForm()
	if formErr != nil {
		c.Error(errs.UserError("Invalid multipart form", http.StatusBadRequest))
		return
	}

	files := form.File["files[]"]
	if len(files) == 0 {
		c.Error(errs.MissingFile)
		return
	}

	scrs, err := h.svc.CreateScreenshot(userID, riceID, files, token.IsAdmin)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"screenshots": scrs})
}

func (h *RiceScreenshotHandler) DeleteScreenshot(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	var path struct {
		RiceID       string `uri:"id" binding:"required,uuid"`
		ScreenshotID string `uri:"screenshotId" binding:"required,uuid"`
	}

	if err := c.ShouldBindUri(&path); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "RiceID") {
			msg = errs.InvalidRiceID.Error()
		} else if strings.Contains(msg, "ScreenshotID") {
			msg = "Invalid screenshot ID path parameter. It must be a valid UUID."
		}
		c.Error(errs.UserError(msg, http.StatusBadRequest))
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)
	screenshotID, _ := uuid.Parse(path.ScreenshotID)

	if err := h.svc.DeleteScreenshot(riceID, screenshotID, userID, token.IsAdmin); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
