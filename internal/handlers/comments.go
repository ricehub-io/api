package handlers

import (
	"errors"
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"ricehub/internal/validation"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type commentsPath struct {
	CommentID string `uri:"id" binding:"required,uuid"`
}

func checkCanUserModifyComment(token *security.AccessToken, commentID string) error {
	if token.IsAdmin {
		return nil
	}

	isAuthor, err := repository.UserOwnsComment(commentID, token.Subject)
	if err != nil || !isAuthor {
		return errs.NoAccess
	}

	return nil
}

func AddComment(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var body models.AddCommentDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	comment, err := repository.InsertComment(body.RiceID, token.Subject, body.Content)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			c.Error(errs.RiceNotFound)
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, comment.ToDTO())
}

func GetRecentComments(c *gin.Context) {
	var query struct {
		Limit int `form:"limit,default=20"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(errs.UserError("Failed to parse limit query parameter", http.StatusBadRequest))
		return
	}

	comments, err := repository.FetchRecentComments(query.Limit)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.CommentsWithUserToDTO(comments))
}

func GetCommentByID(c *gin.Context) {
	var path commentsPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidCommentID)
		return
	}

	comment, err := repository.FindCommentByID(path.CommentID)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.UserError(
			"Comment with provided ID not found",
			http.StatusNotFound,
		)))
		return
	}

	c.JSON(http.StatusOK, comment.ToDTO())
}

func UpdateComment(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path commentsPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidCommentID)
		return
	}

	var update models.UpdateCommentDTO
	if err := validation.ValidateJSON(c, &update); err != nil {
		c.Error(err)
		return
	}

	if err := checkCanUserModifyComment(token, path.CommentID); err != nil {
		c.Error(err)
		return
	}

	comment, err := repository.UpdateComment(path.CommentID, update.Content)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, comment.ToDTO())
}

func DeleteComment(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path commentsPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidCommentID)
		return
	}

	if err := checkCanUserModifyComment(token, path.CommentID); err != nil {
		c.Error(err)
		return
	}

	if err := repository.DeleteComment(path.CommentID); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}
