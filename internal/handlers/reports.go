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

	reportID, err := h.svc.CreateReport(c.Request.Context(), userID, body.RiceID, body.CommentID, body.Reason)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"reportId": reportID})
}

func (h *ReportHandler) ListReports(c *gin.Context) {
	reports, err := h.svc.ListReports(c.Request.Context())
	if err != nil {
		c.Error(err)
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
	reportID, _ := uuid.Parse(path.ReportID)

	report, err := h.svc.GetReportByID(c.Request.Context(), reportID)
	if err != nil {
		c.Error(err)
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
	reportID, _ := uuid.Parse(path.ReportID)

	if err := h.svc.CloseReport(c.Request.Context(), reportID); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
