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

// @Summary Submit a report for a rice or comment
// @Tags reports
// @Accept json
// @Produce json
// @Param body body models.CreateReportDTO true "Report details (riceId OR commentId)"
// @Success 201 {object} object "Returns reportId (UUID)"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 409 {object} models.ErrorDTO "Already reported"
// @Security BearerAuth
// @Router /reports [post]
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

// @Summary List all reports (admin only)
// @Tags reports
// @Produce json
// @Success 200 {array} models.ReportWithUserDTO
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Security BearerAuth
// @Router /reports [get]
func (h *ReportHandler) ListReports(c *gin.Context) {
	reports, err := h.svc.ListReports(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, models.ReportsToDTO(reports))
}

// @Summary Get a report by ID (admin only)
// @Tags reports
// @Produce json
// @Param id path string true "Report ID (UUID)"
// @Success 200 {object} models.ReportWithUserDTO
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Failure 404 {object} models.ErrorDTO "Report not found"
// @Security BearerAuth
// @Router /reports/{id} [get]
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

// @Summary Close a report (admin only)
// @Tags reports
// @Param id path string true "Report ID (UUID)"
// @Success 204 "Closed"
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Failure 404 {object} models.ErrorDTO "Report not found"
// @Security BearerAuth
// @Router /reports/{id}/close [post]
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
