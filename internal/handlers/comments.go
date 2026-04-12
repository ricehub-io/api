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

type CommentHandler struct {
	svc *services.CommentService
}

func NewCommentHandler(svc *services.CommentService) *CommentHandler {
	return &CommentHandler{svc}
}

type commentsPath struct {
	CommentID string `uri:"id" binding:"required,uuid"`
}

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
