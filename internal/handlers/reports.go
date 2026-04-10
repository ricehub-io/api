package handlers

import (
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"ricehub/internal/services"
	"ricehub/internal/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReportHandler struct {
	svc *services.ReportService
}

func NewReportHandler(svc *services.ReportService) *ReportHandler {
	return &ReportHandler{svc}
}

type reportsPath struct {
	ReportID string `uri:"id" binding:"required,uuid"`
}

func (h *ReportHandler) CreateReport(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var body models.CreateReportDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	reportID, err := h.svc.CreateReport(userID, body.RiceID, body.CommentID, body.Reason)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"reportId": reportID})
}

func (h *ReportHandler) ListReports(c *gin.Context) {
	reports, err := repository.FetchReports()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.ReportsToDTO(reports))
}

func (h *ReportHandler) GetReportByID(c *gin.Context) {
	var path reportsPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidReportID)
		return
	}

	report, err := repository.FindReportByID(path.ReportID)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.ReportNotFound))
		return
	}

	c.JSON(http.StatusOK, report.ToDTO())
}

func (h *ReportHandler) CloseReport(c *gin.Context) {
	var path reportsPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidReportID)
		return
	}

	updated, err := repository.CloseReport(path.ReportID, true)
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
