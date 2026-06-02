package models

import "time"

// ExportPeriod defines the time window for a PDF export.
type ExportPeriod string

const (
	ExportPeriodMonthly ExportPeriod = "monthly" // last 30 days
	ExportPeriodYearly  ExportPeriod = "yearly"  // last 365 days
)

// ExportEntrySummary is a single entry's data included in the PDF.
type ExportEntrySummary struct {
	Date      time.Time
	Summary   string
	MoodScore int
	Topics    []string
	KeyQuote  string // single most prominent quote
}

// ExportData is the full dataset used to render the PDF.
type ExportData struct {
	UserName    string
	Period      ExportPeriod
	Since       time.Time
	Until       time.Time
	AvgMood     *int
	PrevAvgMood *int
	MoodDelta   *int
	EntryCount  int
	TopEmotions []string
	DailyMoods  []*DailyMood
	Entries     []*ExportEntrySummary
}
