package handlers

import (
	"net/http"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/security"
	"github.com/ricehub-io/api/internal/services"
	"github.com/ricehub-io/api/internal/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RiceDotfilesHandler struct {
	svc *services.RiceDotfilesService
}

func NewRiceDotfilesHandler(svc *services.RiceDotfilesService) *RiceDotfilesHandler {
	return &RiceDotfilesHandler{svc}
}

// @Summary Purchase dotfiles for a rice
// @Tags rices
// @Produce json
// @Param id path string true "Rice ID (UUID)"
// @Success 200 {object} object "Returns checkoutUrl (string)"
// @Failure 400 {object} models.ErrorDTO "Free dotfiles cannot be purchased"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Failure 409 {object} models.ErrorDTO "Already owned"
// @Security BearerAuth
// @Router /rices/{id}/purchase [post]
func (h *RiceDotfilesHandler) PurchaseDotfiles(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	checkoutURL, err := h.svc.PurchaseDotfiles(c.Request.Context(), userID, riceID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"checkoutUrl": checkoutURL})
}

// @Summary Download dotfiles for a rice
// @Description Returns the dotfiles archive as a file attachment. Free dotfiles are public; paid dotfiles require ownership
// @Tags rices
// @Produce octet-stream
// @Param id path string true "Rice ID (UUID)"
// @Success 200 "File attachment"
// @Failure 403 {object} models.ErrorDTO "Access denied"
// @Failure 404 {object} models.ErrorDTO "Rice not found"
// @Router /rices/{id}/dotfiles [get]
func (h *RiceDotfilesHandler) DownloadDotfiles(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)
	userID := GetUserIDFromRequest(c)

	res, err := h.svc.DownloadDotfiles(c.Request.Context(), riceID, userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.FileAttachment(res.FilePath, res.FileName)
}

// @Summary Replace the dotfiles archive for a rice
// @Tags rices
// @Accept mpfd
// @Produce json
// @Param id path string true "Rice ID (UUID)"
// @Param file formData file true "Dotfiles archive"
// @Success 200 {object} models.RiceDotfilesDTO
// @Failure 400 {object} models.ErrorDTO "Missing file"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /rices/{id}/dotfiles [post]
func (h *RiceDotfilesHandler) UpdateDotfiles(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	file, fileErr := c.FormFile("file")
	if fileErr != nil {
		c.Error(errs.MissingFile)
		return
	}

	df, err := h.svc.UpdateDotfiles(c.Request.Context(), riceID, userID, token.IsAdmin, file)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, df.ToDTO())
}

// @Summary Update the dotfiles type (free or one-time)
// @Tags rices
// @Accept json
// @Param id path string true "Rice ID (UUID)"
// @Param body body models.UpdateDotfilesTypeDTO true "New type"
// @Success 200 "Updated"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /rices/{id}/dotfiles/type [patch]
func (h *RiceDotfilesHandler) UpdateDotfilesType(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	var body models.UpdateDotfilesTypeDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.UpdateDotfilesType(c.Request.Context(), riceID, userID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

// @Summary Update the dotfiles price
// @Tags rices
// @Accept json
// @Param id path string true "Rice ID (UUID)"
// @Param body body models.UpdateDotfilesPriceDTO true "New price"
// @Success 200 "Updated"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /rices/{id}/dotfiles/price [patch]
func (h *RiceDotfilesHandler) UpdateDotfilesPrice(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	var body models.UpdateDotfilesPriceDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.UpdateDotfilesPrice(c.Request.Context(), riceID, userID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
