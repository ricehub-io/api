package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/repository"
	"github.com/ricehub-io/api/internal/security"
	"github.com/ricehub-io/api/internal/services"
	"github.com/ricehub-io/api/internal/validation"

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

// @Summary Create a new rice
// @Tags rices
// @Accept mpfd
// @Param title formData string true "Title (4-32 chars)"
// @Param description formData string true "Description (4-10240 chars)"
// @Param tags formData string false "JSON array of tag IDs e.g. [1,2]"
// @Param dotfilesType formData string false "free or one-time"
// @Param dotfilesPrice formData number false "Price in USD (required when dotfilesType is one-time)"
// @Param dotfiles formData file true "Dotfiles archive"
// @Param "screenshots[]" formData file false "Screenshots (multiple allowed)"
// @Success 201 "Created"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /rices [post]
func (h *RiceHandler) CreateRice(c *gin.Context) {
	var err error

	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

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

	if err := h.svc.CreateRice(
		c.Request.Context(), userID, metadata, screenshots,
		formDotfiles[0], token.IsAdmin, tags,
	); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusCreated)
}

// @Summary List rices with cursor-based pagination
// @Description Pass state=waiting (admin only) to list rices pending review
// @Tags rices
// @Produce json
// @Param sort query string false "Sort order: trending, recent, mostDownloads, mostStars (default: trending)"
// @Param state query string false "Filter by state (admin only: waiting)"
// @Param lastId query string false "Cursor: UUID of the last seen rice"
// @Param lastScore query number false "Cursor: score of the last seen rice (required with lastId when sort=trending)"
// @Param lastCreatedAt query string false "Cursor: ISO timestamp (required with lastId when sort=recent)"
// @Param lastDownloads query integer false "Cursor: downloads count (required with lastId when sort=mostDownloads)"
// @Param lastStars query integer false "Cursor: stars count (required with lastId when sort=mostStars)"
// @Param reverse query boolean false "Reverse pagination direction"
// @Success 200 {object} object "Returns pageCount (int) and rices ([]PartialRiceDTO)"
// @Failure 400 {object} models.ErrorDTO "Bad query parameters"
// @Router /rices [get]
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

	ctx := c.Request.Context()
	if query.State == "waiting" {
		if isAdmin {
			rices, err := h.svc.ListWaitingRices(ctx)
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

	res, err := h.svc.ListRices(ctx, query.Sort, pag, userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pageCount": res.PageCount,
		"rices":     res.Rices.ToDTO(),
	})
}

// @Summary Get a rice by ID
// @Tags rices
// @Produce json
// @Param id path string true "Rice ID (UUID)"
// @Success 200 {object} models.RiceWithRelationsDTO
// @Failure 400 {object} models.ErrorDTO "Invalid UUID"
// @Failure 404 {object} models.ErrorDTO "Rice not found"
// @Router /rices/{id} [get]
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

	rice, err := h.svc.GetRiceByID(c.Request.Context(), userID, riceID, isAdmin)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rice.ToDTO())
}

// @Summary List comments for a rice
// @Tags rices
// @Produce json
// @Param id path string true "Rice ID (UUID)"
// @Success 200 {array} models.CommentWithUserDTO
// @Failure 400 {object} models.ErrorDTO "Invalid UUID"
// @Failure 404 {object} models.ErrorDTO "Rice not found"
// @Router /rices/{id}/comments [get]
func (h *RiceHandler) ListRiceComments(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	comments, err := h.svc.ListRiceComments(c.Request.Context(), riceID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, models.CommentsWithUserToDTO(comments))
}

// @Summary Update rice title and/or description
// @Tags rices
// @Accept json
// @Param id path string true "Rice ID (UUID)"
// @Param body body models.UpdateRiceDTO true "Fields to update"
// @Success 200 "Updated"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Failure 404 {object} models.ErrorDTO "Rice not found"
// @Security BearerAuth
// @Router /rices/{id} [patch]
func (h *RiceHandler) UpdateRiceMetadata(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

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

	if err := h.svc.UpdateRiceMetadata(c.Request.Context(), riceID, userID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

// @Summary Accept or reject a waiting rice (admin only)
// @Tags rices
// @Accept json
// @Param id path string true "Rice ID (UUID)"
// @Param body body models.UpdateRiceStateDTO true "New state"
// @Success 200 "Accepted"
// @Success 204 "Rejected"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Failure 404 {object} models.ErrorDTO "Rice not found"
// @Security BearerAuth
// @Router /rices/{id}/state [patch]
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

	rejected, err := h.svc.UpdateRiceState(c.Request.Context(), riceID, body)
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

// @Summary Delete a rice
// @Tags rices
// @Param id path string true "Rice ID (UUID)"
// @Success 204 "Deleted"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Failure 404 {object} models.ErrorDTO "Rice not found"
// @Security BearerAuth
// @Router /rices/{id} [delete]
func (h *RiceHandler) DeleteRice(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	if err := h.svc.DeleteRice(c.Request.Context(), riceID, userID, token.IsAdmin); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
