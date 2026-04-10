package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"ricehub/internal/services"
	"ricehub/internal/validation"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RiceHandler struct {
	svc *services.RiceService
}

func NewRiceHandler(svc *services.RiceService) *RiceHandler {
	return &RiceHandler{svc}
}

type ricesPath struct {
	RiceID string `uri:"id" binding:"required,uuid"`
}

func (h *RiceHandler) CreateRice(c *gin.Context) {
	var err error

	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

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

	if len(formDotfiles) == 0 {
		c.Error(errs.UserError("Dotfiles are required", http.StatusBadRequest))
		return
	}

	tags := []int{}
	if metadata.Tags != "" && metadata.Tags != "[]" {
		if err := json.Unmarshal([]byte(metadata.Tags), &tags); err != nil {
			c.Error(errs.UserError(
				fmt.Sprintf("Failed to parse tags: %v", strings.ReplaceAll(err.Error(), `"`, `'`)),
				http.StatusBadRequest,
			))
			return
		}
	}

	if err := h.svc.CreateRice(userID, metadata, screenshots, formDotfiles[0], token.IsAdmin, tags); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusCreated)
}

func (h *RiceHandler) ListRices(c *gin.Context) {
	token := GetTokenFromRequest(c)
	isAdmin := token != nil && token.IsAdmin

	var query struct {
		Sort          models.SortBy `form:"sort,default=trending"`
		State         string        `form:"state"`
		LastID        *string       `form:"lastId" binding:"omitempty,uuid"`
		LastScore     *float32      `form:"lastScore"`
		LastCreatedAt *time.Time    `form:"lastCreatedAt"`
		LastStars     *int          `form:"lastStars"`
		LastDownloads *int          `form:"lastDownloads"`
		Reverse       bool          `form:"reverse"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(errs.UserError(
			fmt.Sprintf("Failed to parse query parameters: %v", strings.ReplaceAll(err.Error(), `"`, `'`)),
			http.StatusBadRequest,
		))
		return
	}

	if query.State == "waiting" {
		if isAdmin {
			rices, err := h.svc.ListWaitingRices()
			if err != nil {
				c.Error(err)
				return
			}
			c.JSON(http.StatusOK, rices.ToDTO())
		} else {
			c.Error(errs.NoAccess)
		}
		return
	}

	hasCursor := query.LastScore != nil || query.LastCreatedAt != nil || query.LastDownloads != nil || query.LastStars != nil
	if hasCursor && query.LastID == nil {
		c.Error(errs.UserError("Cursor fields require lastId", http.StatusBadRequest))
		return
	}
	if query.LastID != nil {
		cursorMissing := false
		switch query.Sort {
		case models.Trending:
			cursorMissing = query.LastScore == nil
		case models.Recent:
			cursorMissing = query.LastCreatedAt == nil
		case models.MostDownloads:
			cursorMissing = query.LastDownloads == nil
		case models.MostStars:
			cursorMissing = query.LastStars == nil
		}
		if cursorMissing {
			c.Error(errs.UserError(
				"lastId requires a matching cursor field for the selected sort",
				http.StatusBadRequest,
			))
			return
		}
	}

	pag := repository.Pagination{
		LastID:        query.LastID,
		LastScore:     query.LastScore,
		LastCreatedAt: query.LastCreatedAt,
		LastDownloads: query.LastDownloads,
		Reverse:       query.Reverse,
	}

	var userID *uuid.UUID
	if token != nil {
		tmp, _ := uuid.Parse(token.Subject)
		userID = &tmp
	}

	res, err := h.svc.ListRices(query.Sort, pag, userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pageCount": res.PageCount,
		"rices":     res.Rices.ToDTO(),
	})
}

func (h *RiceHandler) GetRiceByID(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	token := GetTokenFromRequest(c)
	isAdmin := token != nil && token.IsAdmin
	userID := GetUserIDFromRequest(c)

	rice, err := h.svc.GetRiceByID(userID, riceID, isAdmin)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rice.ToDTO())
}

func (h *RiceHandler) ListRiceComments(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	comments, err := h.svc.ListRiceComments(riceID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, models.CommentsWithUserToDTO(comments))
}

func (h *RiceHandler) UpdateRiceMetadata(c *gin.Context) {
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

	var body models.UpdateRiceDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.UpdateRiceMetadata(riceID, userID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *RiceHandler) UpdateRiceState(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	var body models.UpdateRiceStateDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	rejected, err := h.svc.UpdateRiceState(riceID, body)
	if err != nil {
		c.Error(err)
		return
	}

	if rejected {
		c.Status(http.StatusNoContent)
	} else {
		c.Status(http.StatusOK)
	}
}

func (h *RiceHandler) DeleteRice(c *gin.Context) {
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

	if err := h.svc.DeleteRice(riceID, userID, token.IsAdmin); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
