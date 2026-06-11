package handlers

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	fpdf "github.com/go-pdf/fpdf"
	"github.com/google/uuid"
)

type exportRepo interface {
	ExportData(ctx context.Context, userID uuid.UUID, since, until time.Time) (*models.ExportData, error)
}

type userNameRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type ExportHandler struct {
	analysisRepo exportRepo
	userRepo     userNameRepo
}

func NewExportHandler(analysisRepo exportRepo, userRepo userNameRepo) *ExportHandler {
	return &ExportHandler{analysisRepo: analysisRepo, userRepo: userRepo}
}

// GET /export/pdf?period=monthly|yearly
func (h *ExportHandler) ExportPDF(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		_ = c.Error(apierr.Unauthorized("user not found"))
		return
	}
	if !user.EffectivePlan().AtLeast(models.PlanPlus) {
		_ = c.Error(apierr.Forbidden("PDF export requires DreamLog+ or higher"))
		return
	}
	userID := user.ID
	periodStr := c.DefaultQuery("period", "monthly")

	var period models.ExportPeriod
	var days int
	switch periodStr {
	case "yearly":
		period = models.ExportPeriodYearly
		days = 365
	default:
		period = models.ExportPeriodMonthly
		days = 30
	}

	until := time.Now().UTC()
	since := until.AddDate(0, 0, -days)

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to load user"))
		return
	}

	data, err := h.analysisRepo.ExportData(c.Request.Context(), userID, since, until)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to load export data"))
		return
	}

	data.Period = period
	data.UserName = user.Name
	if user.PreferredName != nil && *user.PreferredName != "" {
		data.UserName = *user.PreferredName
	}

	pdf := buildPDF(data)

	filename := fmt.Sprintf("dreamlog-%s-%s.pdf", periodStr, until.Format("2006-01"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "application/pdf")

	if err := pdf.Output(c.Writer); err != nil {
		_ = c.Error(apierr.Internal("failed to write pdf"))
	}
}

// ── PDF construction ─────────────────────────────────────────────────────────

const (
	pageW  = 210.0 // A4 width mm
	pageH  = 297.0 // A4 height mm
	margin = 18.0
	bodyW  = pageW - 2*margin

	colorBg      = "#1a1625"
	colorPurple  = "#7c5cbf"
	colorLight   = "#e8e0f5"
	colorMuted   = "#9b8ec4"
	colorGreen   = "#4ade80"
	colorYellow  = "#facc15"
	colorOrange  = "#fb923c"
	colorRed     = "#f87171"
	colorWhite   = "#ffffff"
)

func hexRGB(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 6 {
		fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	}
	return
}

func setFill(pdf *fpdf.Fpdf, hex string) {
	r, g, b := hexRGB(hex)
	pdf.SetFillColor(r, g, b)
}

func setDraw(pdf *fpdf.Fpdf, hex string) {
	r, g, b := hexRGB(hex)
	pdf.SetDrawColor(r, g, b)
}

func setTextColor(pdf *fpdf.Fpdf, hex string) {
	r, g, b := hexRGB(hex)
	pdf.SetTextColor(r, g, b)
}

func moodColor(score int) string {
	switch {
	case score >= 71:
		return colorGreen
	case score >= 46:
		return colorYellow
	case score >= 26:
		return colorOrange
	default:
		return colorRed
	}
}

func moodLabel(score int) string {
	switch {
	case score >= 71:
		return "High"
	case score >= 46:
		return "Moderate"
	case score >= 26:
		return "Low"
	default:
		return "Very Low"
	}
}

func buildPDF(data *models.ExportData) *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, margin)

	// Use built-in Helvetica - no external font files needed.
	pdf.SetFont("Helvetica", "", 12)

	// ── Cover page ────────────────────────────────────────────────────────────
	pdf.AddPage()

	// Background rectangle.
	setFill(pdf, colorBg)
	pdf.Rect(0, 0, pageW, pageH, "F")

	// Logo wordmark.
	pdf.SetY(60)
	setTextColor(pdf, colorPurple)
	pdf.SetFont("Helvetica", "B", 32)
	pdf.CellFormat(pageW, 14, "DreamLog", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 13)
	setTextColor(pdf, colorMuted)
	pdf.CellFormat(pageW, 8, "Emotional Journal", "", 1, "C", false, 0, "")

	pdf.Ln(16)

	// Divider.
	setDraw(pdf, colorPurple)
	pdf.SetLineWidth(0.4)
	pdf.Line(margin+30, pdf.GetY(), pageW-margin-30, pdf.GetY())
	pdf.Ln(16)

	// User name.
	setTextColor(pdf, colorLight)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(pageW, 10, data.UserName, "", 1, "C", false, 0, "")
	pdf.Ln(6)

	// Period label.
	pdf.SetFont("Helvetica", "", 12)
	setTextColor(pdf, colorMuted)
	periodLabel := fmt.Sprintf("%s – %s",
		data.Since.Format("2 Jan 2006"),
		data.Until.Format("2 Jan 2006"),
	)
	pdf.CellFormat(pageW, 8, periodLabel, "", 1, "C", false, 0, "")
	pdf.Ln(24)

	// Summary stats row.
	drawStatBox(pdf, "Entries", fmt.Sprintf("%d", data.EntryCount), margin, pdf.GetY())
	avgStr := "-"
	if data.AvgMood != nil {
		avgStr = fmt.Sprintf("%d / 100", *data.AvgMood)
	}
	drawStatBox(pdf, "Avg Mood", avgStr, margin+(bodyW/2), pdf.GetY())
	pdf.Ln(36)

	// Top emotions on cover.
	if len(data.TopEmotions) > 0 {
		setTextColor(pdf, colorMuted)
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(pageW, 6, "TOP EMOTIONS", "", 1, "C", false, 0, "")
		pdf.Ln(4)
		emotionLine := strings.Join(data.TopEmotions, "   ·   ")
		setTextColor(pdf, colorLight)
		pdf.SetFont("Helvetica", "I", 12)
		pdf.CellFormat(pageW, 8, emotionLine, "", 1, "C", false, 0, "")
	}

	// Footer.
	pdf.SetY(pageH - 20)
	setTextColor(pdf, colorMuted)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(pageW, 6, "Generated by DreamLog · dreamlog.app", "", 1, "C", false, 0, "")

	// ── Mood overview page ────────────────────────────────────────────────────
	pdf.AddPage()
	setFill(pdf, colorBg)
	pdf.Rect(0, 0, pageW, pageH, "F")

	sectionHeader(pdf, "Mood Overview")

	// Stats cards.
	pdf.Ln(4)
	y := pdf.GetY()
	cardW := (bodyW - 8) / 3

	drawCard(pdf, margin, y, cardW, "AVG MOOD",
		func() string {
			if data.AvgMood != nil {
				return fmt.Sprintf("%d", *data.AvgMood)
			}
			return "-"
		}(),
		func() string {
			if data.AvgMood != nil {
				return moodLabel(*data.AvgMood)
			}
			return ""
		}(),
		func() string {
			if data.AvgMood != nil {
				return moodColor(*data.AvgMood)
			}
			return colorMuted
		}(),
	)
	drawCard(pdf, margin+cardW+4, y, cardW, "ENTRIES", fmt.Sprintf("%d", data.EntryCount), "total", colorPurple)
	drawCard(pdf, margin+2*(cardW+4), y, cardW, "TREND",
		func() string {
			if data.MoodDelta != nil {
				if *data.MoodDelta >= 0 {
					return fmt.Sprintf("+%d", *data.MoodDelta)
				}
				return fmt.Sprintf("%d", *data.MoodDelta)
			}
			return "-"
		}(),
		func() string {
			if data.MoodDelta != nil && *data.MoodDelta >= 0 {
				return "vs prior period"
			}
			if data.MoodDelta != nil {
				return "vs prior period"
			}
			return "no prior data"
		}(),
		func() string {
			if data.MoodDelta != nil && *data.MoodDelta >= 0 {
				return colorGreen
			}
			return colorOrange
		}(),
	)
	pdf.SetY(y + 32)
	pdf.Ln(8)

	// Mood bar chart.
	sectionSubHeader(pdf, "Daily Mood")
	pdf.Ln(2)
	drawMoodChart(pdf, data.DailyMoods)
	pdf.Ln(8)

	// Top emotions section.
	if len(data.TopEmotions) > 0 {
		sectionSubHeader(pdf, "Top Emotions")
		pdf.Ln(2)
		for i, e := range data.TopEmotions {
			pct := 100 - i*15
			if pct < 25 {
				pct = 25
			}
			drawEmotionBar(pdf, e, pct)
		}
	}

	// ── Entry summaries pages ─────────────────────────────────────────────────
	if len(data.Entries) > 0 {
		pdf.AddPage()
		setFill(pdf, colorBg)
		pdf.Rect(0, 0, pageW, pageH, "F")
		sectionHeader(pdf, "Journal Entries")
		pdf.Ln(4)

		for _, entry := range data.Entries {
			drawEntryCard(pdf, entry)
		}
	}

	return pdf
}

// drawStatBox draws a small stat on the cover page.
func drawStatBox(pdf *fpdf.Fpdf, label, value string, x, y float64) {
	pdf.SetXY(x, y)
	w := bodyW / 2
	setFill(pdf, "#2d2640")
	pdf.RoundedRect(x, y, w, 28, 4, "1234", "F")

	setTextColor(pdf, colorMuted)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetXY(x, y+6)
	pdf.CellFormat(w, 5, strings.ToUpper(label), "", 1, "C", false, 0, "")

	setTextColor(pdf, colorLight)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(x, y+13)
	pdf.CellFormat(w, 9, value, "", 1, "C", false, 0, "")
}

// drawCard draws a metric card.
func drawCard(pdf *fpdf.Fpdf, x, y, w float64, label, value, sublabel, accentHex string) {
	h := 30.0
	setFill(pdf, "#2d2640")
	pdf.RoundedRect(x, y, w, h, 3, "1234", "F")

	setTextColor(pdf, colorMuted)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(x+3, y+5)
	pdf.CellFormat(w-6, 4, label, "", 1, "L", false, 0, "")

	r, g, b := hexRGB(accentHex)
	pdf.SetTextColor(r, g, b)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(x+3, y+11)
	pdf.CellFormat(w-6, 8, value, "", 1, "L", false, 0, "")

	setTextColor(pdf, colorMuted)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(x+3, y+22)
	pdf.CellFormat(w-6, 4, sublabel, "", 1, "L", false, 0, "")
}

// sectionHeader renders a bold section title.
func sectionHeader(pdf *fpdf.Fpdf, title string) {
	setTextColor(pdf, colorLight)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(bodyW, 10, title, "", 1, "L", false, 0, "")
	setDraw(pdf, colorPurple)
	pdf.SetLineWidth(0.3)
	pdf.Line(margin, pdf.GetY(), margin+bodyW, pdf.GetY())
	pdf.Ln(4)
}

func sectionSubHeader(pdf *fpdf.Fpdf, title string) {
	setTextColor(pdf, colorMuted)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(bodyW, 6, strings.ToUpper(title), "", 1, "L", false, 0, "")
}

// drawMoodChart renders a row of daily mood bars.
func drawMoodChart(pdf *fpdf.Fpdf, days []*models.DailyMood) {
	if len(days) == 0 {
		setTextColor(pdf, colorMuted)
		pdf.SetFont("Helvetica", "I", 10)
		pdf.CellFormat(bodyW, 8, "No entries in this period.", "", 1, "L", false, 0, "")
		return
	}

	maxBars := 62
	if len(days) > maxBars {
		days = days[len(days)-maxBars:]
	}

	chartH := 28.0
	barW := math.Max(1.5, bodyW/float64(len(days))-0.5)
	gap := (bodyW - float64(len(days))*barW) / float64(len(days)+1)
	if gap < 0.3 {
		gap = 0.3
	}

	startY := pdf.GetY()
	startX := margin

	// Chart background.
	setFill(pdf, "#2d2640")
	pdf.RoundedRect(startX-2, startY, bodyW+4, chartH+4, 3, "1234", "F")

	for i, dm := range days {
		barH := float64(dm.AvgMood) / 100.0 * (chartH - 4)
		if barH < 1 {
			barH = 1
		}
		x := startX + float64(i)*(barW+gap)
		y := startY + chartH - barH

		setFill(pdf, moodColor(dm.AvgMood))
		pdf.RoundedRect(x, y, barW, barH, 0.8, "1234", "F")
	}

	// Axis labels: first and last date.
	pdf.SetY(startY + chartH + 6)
	setTextColor(pdf, colorMuted)
	pdf.SetFont("Helvetica", "", 7)
	first := days[0].Day
	last := days[len(days)-1].Day
	pdf.CellFormat(bodyW/2, 4, first, "", 0, "L", false, 0, "")
	pdf.CellFormat(bodyW/2, 4, last, "", 1, "R", false, 0, "")
}

// drawEmotionBar renders an emotion with a visual fill bar.
func drawEmotionBar(pdf *fpdf.Fpdf, emotion string, pct int) {
	pdf.Ln(1)
	y := pdf.GetY()
	barH := 7.0
	fillW := bodyW * float64(pct) / 100.0

	setFill(pdf, "#2d2640")
	pdf.RoundedRect(margin, y, bodyW, barH, 2, "1234", "F")
	setFill(pdf, colorPurple)
	pdf.RoundedRect(margin, y, fillW, barH, 2, "1234", "F")

	setTextColor(pdf, colorLight)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(margin+3, y+1.5)
	pdf.CellFormat(bodyW-6, barH-3, strings.Title(emotion), "", 1, "L", false, 0, "") //nolint:staticcheck
}

// drawEntryCard renders a single journal entry summary.
func drawEntryCard(pdf *fpdf.Fpdf, entry *models.ExportEntrySummary) {
	// Check remaining page space; 40mm minimum.
	if pdf.GetY() > pageH-margin-40 {
		pdf.AddPage()
		setFill(pdf, colorBg)
		pdf.Rect(0, 0, pageW, pageH, "F")
		pdf.SetY(margin)
	}

	y := pdf.GetY()
	cardH := estimateCardHeight(pdf, entry)

	// Check if card fits.
	if y+cardH > pageH-margin {
		pdf.AddPage()
		setFill(pdf, colorBg)
		pdf.Rect(0, 0, pageW, pageH, "F")
		pdf.SetY(margin)
		y = margin
	}

	setFill(pdf, "#2d2640")
	pdf.RoundedRect(margin, y, bodyW, cardH, 3, "1234", "F")

	// Date + mood score badge.
	pdf.SetXY(margin+4, y+5)
	setTextColor(pdf, colorMuted)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(bodyW-50, 5, entry.Date.Format("Monday, 2 January 2006"), "", 0, "L", false, 0, "")

	// Mood badge.
	mc := moodColor(entry.MoodScore)
	br, bg, bb := hexRGB(mc)
	pdf.SetFillColor(br, bg, bb)
	badgeX := margin + bodyW - 28
	pdf.RoundedRect(badgeX, y+3, 24, 8, 2, "1234", "F")
	setTextColor(pdf, colorBg)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetXY(badgeX, y+4)
	pdf.CellFormat(24, 6, fmt.Sprintf("%d", entry.MoodScore), "", 1, "C", false, 0, "")

	// Summary.
	pdf.SetXY(margin+4, y+13)
	setTextColor(pdf, colorLight)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(bodyW-8, 5, entry.Summary, "", "L", false)

	// Key quote.
	if entry.KeyQuote != "" {
		pdf.SetX(margin + 8)
		setTextColor(pdf, colorMuted)
		pdf.SetFont("Helvetica", "I", 9)
		pdf.MultiCell(bodyW-16, 4.5, "\""+entry.KeyQuote+"\"", "", "L", false)
	}

	// Topics.
	if len(entry.Topics) > 0 {
		pdf.SetX(margin + 4)
		setTextColor(pdf, colorPurple)
		pdf.SetFont("Helvetica", "", 8)
		topics := strings.Join(entry.Topics, " · ")
		pdf.CellFormat(bodyW-8, 4, topics, "", 1, "L", false, 0, "")
	}

	pdf.SetY(y + cardH + 4)
}

func estimateCardHeight(pdf *fpdf.Fpdf, entry *models.ExportEntrySummary) float64 {
	// Rough estimate: header + summary lines + quote + topics + padding.
	lines := float64(len([]rune(entry.Summary))) / 80.0
	if lines < 1 {
		lines = 1
	}
	h := 13.0 + lines*5.5 + 6
	if entry.KeyQuote != "" {
		qlines := float64(len([]rune(entry.KeyQuote)))/70.0 + 1
		h += qlines * 4.5
	}
	if len(entry.Topics) > 0 {
		h += 5
	}
	return math.Ceil(h)
}
