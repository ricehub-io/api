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

type CommentHandler struct {
	svc *services.CommentService
}

func NewCommentHandler(svc *services.CommentService) *CommentHandler {
	return &CommentHandler{svc}
}

type commentsPath struct {
	CommentID string `uri:"id" binding:"required,uuid"`
}

// @Summary Create a comment on a rice
// @Tags comments
// @Accept json
// @Produce json
// @Param body body models.CreateCommentDTO true "Comment content and rice ID"
// @Success 201 {object} models.RiceCommentDTO
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 404 {object} models.ErrorDTO "Rice not found"
// @Security BearerAuth
// @Router /comments [post]
func (h *CommentHandler) CreateComment(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var body models.CreateCommentDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	comment, err := h.svc.CreateComment(c.Request.Context(), userID, body)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, comment.ToDTO())
}

// @Summary List recent comments (admin only)
// @Tags comments
// @Produce json
// @Param limit query int false "Max number of comments to return (default 20)"
// @Success 200 {array} models.CommentWithUserDTO
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Security BearerAuth
// @Router /comments [get]
func (h *CommentHandler) ListComments(c *gin.Context) {
	var query struct {
		Limit int `form:"limit,default=20" binding:"gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(errs.UserError("Failed to parse limit query parameter", http.StatusBadRequest))
		return
	}

	comments, err := h.svc.ListComments(c.Request.Context(), query.Limit)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, models.CommentsWithUserToDTO(comments))
}

// @Summary Get a comment by ID
// @Tags comments
// @Produce json
// @Param id path string true "Comment ID (UUID)"
// @Success 200 {object} models.RiceCommentWithSlugDTO
// @Failure 400 {object} models.ErrorDTO "Invalid UUID"
// @Failure 404 {object} models.ErrorDTO "Comment not found"
// @Security BearerAuth
// @Router /comments/{id} [get]
func (h *CommentHandler) GetCommentByID(c *gin.Context) {
	var path commentsPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidCommentID)
		return
	}
	commentID, _ := uuid.Parse(path.CommentID)

	comment, err := h.svc.GetCommentByID(c.Request.Context(), commentID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, comment.ToDTO())
}

// @Summary Update a comment's content
// @Tags comments
// @Accept json
// @Produce json
// @Param id path string true "Comment ID (UUID)"
// @Param body body models.UpdateCommentDTO true "New content"
// @Success 200 {object} models.RiceCommentDTO
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Failure 404 {object} models.ErrorDTO "Comment not found"
// @Security BearerAuth
// @Router /comments/{id} [patch]
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var path commentsPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidCommentID)
		return
	}

	var body models.UpdateCommentDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}
	commentID, _ := uuid.Parse(path.CommentID)

	comment, err := h.svc.UpdateComment(c.Request.Context(), token.IsAdmin, userID, commentID, body.Content)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, comment.ToDTO())
}

// @Summary Delete a comment
// @Tags comments
// @Param id path string true "Comment ID (UUID)"
// @Success 204 "Deleted"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Failure 404 {object} models.ErrorDTO "Comment not found"
// @Security BearerAuth
// @Router /comments/{id} [delete]
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var path commentsPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidCommentID)
		return
	}
	commentID, _ := uuid.Parse(path.CommentID)

	if err := h.svc.DeleteComment(c.Request.Context(), token.IsAdmin, userID, commentID); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
