package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InsightShareRepository struct {
	db *pgxpool.Pool
}

func NewInsightShareRepository(db *pgxpool.Pool) *InsightShareRepository {
	return &InsightShareRepository{db: db}
}

// RecordShare inserts a share event and returns the updated all-time share count.
func (r *InsightShareRepository) RecordShare(ctx context.Context, userID uuid.UUID, weekStart string) (*models.InsightShare, error) {
	const q = `
		INSERT INTO insight_shares (user_id, week_start)
		VALUES ($1, $2::DATE)
		RETURNING id, user_id, week_start::TEXT, created_at`

	var s models.InsightShare
	err := r.db.QueryRow(ctx, q, userID, weekStart).Scan(&s.ID, &s.UserID, &s.WeekStart, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insightShares.RecordShare: %w", err)
	}
	return &s, nil
}

// CountByUser returns the total number of insight cards shared by the user.
func (r *InsightShareRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM insight_shares WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("insightShares.CountByUser: %w", err)
	}
	return count, nil
}

// GetCardData returns everything needed to render the InsightCard for the current week.
// weekStart must be a Monday in YYYY-MM-DD format.
func (r *InsightShareRepository) GetCardData(ctx context.Context, userID uuid.UUID, weekStart time.Time) (*models.InsightCardData, error) {
	weekStartStr := weekStart.Format("2006-01-02")

	// Daily mood arc Mon–Sun for the current week.
	const arcQ = `
		SELECT TO_CHAR(DATE(e.created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day,
		       ROUND(AVG(ea.mood_score))::int AS avg_mood
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND DATE(e.created_at AT TIME ZONE 'UTC') >= $2::DATE
		  AND DATE(e.created_at AT TIME ZONE 'UTC') < ($2::DATE + INTERVAL '7 days')
		GROUP BY day
		ORDER BY day ASC`

	arcRows, err := r.db.Query(ctx, arcQ, userID, weekStartStr)
	if err != nil {
		return nil, fmt.Errorf("insightShares.GetCardData arc: %w", err)
	}
	defer arcRows.Close()

	var arc []models.MoodArcDay
	for arcRows.Next() {
		var d models.MoodArcDay
		if err := arcRows.Scan(&d.Date, &d.AvgMood); err != nil {
			return nil, fmt.Errorf("insightShares.GetCardData arc scan: %w", err)
		}
		arc = append(arc, d)
	}
	if err := arcRows.Err(); err != nil {
		return nil, fmt.Errorf("insightShares.GetCardData arc rows: %w", err)
	}

	// Top 3 emotions this week.
	const emoQ = `
		SELECT em.emotion_text, COUNT(*) AS cnt
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id,
		     LATERAL jsonb_array_elements(ea.emotional_tone::jsonb) AS et,
		     LATERAL (SELECT et->>'emotion' AS emotion_text) AS em
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND DATE(e.created_at AT TIME ZONE 'UTC') >= $2::DATE
		  AND DATE(e.created_at AT TIME ZONE 'UTC') < ($2::DATE + INTERVAL '7 days')
		  AND (et->>'intensity')::float >= 0.3
		GROUP BY em.emotion_text
		ORDER BY cnt DESC
		LIMIT 3`

	emoRows, err := r.db.Query(ctx, emoQ, userID, weekStartStr)
	if err != nil {
		return nil, fmt.Errorf("insightShares.GetCardData emotions: %w", err)
	}
	defer emoRows.Close()

	var topEmotions []string
	for emoRows.Next() {
		var em string
		var cnt int
		if err := emoRows.Scan(&em, &cnt); err != nil {
			return nil, fmt.Errorf("insightShares.GetCardData emotions scan: %w", err)
		}
		topEmotions = append(topEmotions, em)
	}

	// Entry count this week.
	var entryCount int
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM entries
		WHERE user_id = $1
		  AND status = 'completed'
		  AND DATE(created_at AT TIME ZONE 'UTC') >= $2::DATE
		  AND DATE(created_at AT TIME ZONE 'UTC') < ($2::DATE + INTERVAL '7 days')`,
		userID, weekStartStr,
	).Scan(&entryCount)
	if err != nil {
		return nil, fmt.Errorf("insightShares.GetCardData entryCount: %w", err)
	}

	// All-time share count.
	shareCount, err := r.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	weekEnd := weekStart.AddDate(0, 0, 6)
	weekLabel := fmt.Sprintf("%s – %s", weekStart.Format("Jan 2"), weekEnd.Format("Jan 2"))

	return &models.InsightCardData{
		WeekLabel:   weekLabel,
		WeekStart:   weekStartStr,
		MoodArc:     arc,
		TopEmotions: topEmotions,
		EntryCount:  entryCount,
		ShareCount:  shareCount,
	}, nil
}

// CurrentWeekStart returns the Monday of the current ISO week in UTC.
func CurrentWeekStart() time.Time {
	now := time.Now().UTC()
	wd := int(now.Weekday()) // 0=Sunday
	if wd == 0 {
		wd = 7 // treat Sunday as day 7 so Monday = 1
	}
	daysBack := wd - 1
	monday := now.AddDate(0, 0, -daysBack)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}
