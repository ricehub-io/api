package handlers

import (
	"errors"
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"
	"ricehub/src/security"
	"ricehub/src/utils"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func FetchReports(c *gin.Context) {
	reports, err := repository.FetchReports()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.ReportsToDTO(reports))
}

func GetReportByID(c *gin.Context) {
	reportID := c.Param("reportId")
	report, err := repository.FindReportByID(reportID)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.ReportNotFound))
		return
	}

	c.JSON(http.StatusOK, report.ToDTO())
}

func CreateReport(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)

	var report models.CreateReportDTO
	if err := utils.ValidateJSON(c, &report); err != nil {
		c.Error(err)
		return
	}

	if report.RiceID != nil && report.CommentID != nil {
		c.Error(errs.UserError("Too many resources provided! You can only report one thing at a time.", http.StatusBadRequest))
		return
	}

	reportID, err := repository.InsertReport(token.Subject, report.Reason, report.RiceID, report.CommentID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.ForeignKeyViolation:
				c.Error(errs.UserError("Resource with provided ID not found!", http.StatusNotFound))
				return
			case pgerrcode.UniqueViolation:
				c.Error(errs.UserError("You have already submitted a similar report!", http.StatusConflict))
				return
			}
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"reportId": reportID})
}

func CloseReport(c *gin.Context) {
	reportID := c.Param("reportId")

	updated, err := repository.CloseReport(reportID, true)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !updated {
		c.Error(errs.ReportNotFound)
		return
	}

	c.Status(http.StatusNoContent)
}
