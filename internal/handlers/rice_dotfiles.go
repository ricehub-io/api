package handlers

import (
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/security"
	"ricehub/internal/services"
	"ricehub/internal/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RiceDotfilesHandler struct {
	svc *services.RiceDotfilesService
}

func NewRiceDotfilesHandler(svc *services.RiceDotfilesService) *RiceDotfilesHandler {
	return &RiceDotfilesHandler{svc}
}

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
