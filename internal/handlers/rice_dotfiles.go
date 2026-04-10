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

func PurchaseDotfiles(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	checkoutURL, err := services.PurchaseDotfiles(userID, riceID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"checkoutUrl": checkoutURL})
}

func DownloadDotfiles(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)
	userID := GetUserIDFromRequest(c)

	res, err := services.DownloadDotfiles(riceID, userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.FileAttachment(res.FilePath, res.FileName)
}

func UpdateDotfiles(c *gin.Context) {
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

	file, fileErr := c.FormFile("file")
	if fileErr != nil {
		c.Error(errs.MissingFile)
		return
	}

	df, err := services.UpdateDotfiles(riceID, userID, token.IsAdmin, file)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, df.ToDTO())
}

func UpdateDotfilesType(c *gin.Context) {
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

	var body models.UpdateDotfilesTypeDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := services.UpdateDotfilesType(riceID, userID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func UpdateDotfilesPrice(c *gin.Context) {
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

	var body models.UpdateDotfilesPriceDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := services.UpdateDotfilesPrice(riceID, userID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
