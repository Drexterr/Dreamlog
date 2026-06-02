package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LifeChapterRepository struct {
	db *pgxpool.Pool
}

func NewLifeChapterRepository(db *pgxpool.Pool) *LifeChapterRepository {
	return &LifeChapterRepository{db: db}
}

// Create inserts a new chapter and returns it.
func (r *LifeChapterRepository) Create(ctx context.Context, userID uuid.UUID, input models.CreateChapterInput) (*models.LifeChapter, error) {
	color := input.Color
	if color == "" {
		color = "#7C3AED"
	}

	const q = `
		INSERT INTO life_chapters (user_id, title, description, start_date, end_date, emoji, color)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, title, description, start_date, end_date, emoji, color, summary, created_at, updated_at`

	row := r.db.QueryRow(ctx, q,
		userID,
		input.Title,
		input.Description,
		input.StartDate,
		input.EndDate,
		input.Emoji,
		color,
	)
	return scanChapter(row)
}

// List returns all chapters for a user, newest start_date first.
func (r *LifeChapterRepository) List(ctx context.Context, userID uuid.UUID) ([]*models.LifeChapter, error) {
	const q = `
		SELECT id, user_id, title, description, start_date, end_date, emoji, color, summary, created_at, updated_at
		FROM life_chapters
		WHERE user_id = $1
		ORDER BY start_date DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("lifeChapters.List: %w", err)
	}
	defer rows.Close()

	var chapters []*models.LifeChapter
	for rows.Next() {
		ch, err := scanChapter(rows)
		if err != nil {
			return nil, fmt.Errorf("lifeChapters.List scan: %w", err)
		}
		chapters = append(chapters, ch)
	}
	return chapters, rows.Err()
}

// GetByID returns a single chapter owned by userID, or nil if not found.
func (r *LifeChapterRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*models.LifeChapter, error) {
	const q = `
		SELECT id, user_id, title, description, start_date, end_date, emoji, color, summary, created_at, updated_at
		FROM life_chapters
		WHERE id = $1 AND user_id = $2`

	ch, err := scanChapter(r.db.QueryRow(ctx, q, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lifeChapters.GetByID: %w", err)
	}
	return ch, nil
}

// Update applies the given non-nil fields to the chapter and returns the updated row.
func (r *LifeChapterRepository) Update(ctx context.Context, id, userID uuid.UUID, input models.UpdateChapterInput) (*models.LifeChapter, error) {
	// Fetch current to merge.
	ch, err := r.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, nil
	}

	if input.Title != nil {
		ch.Title = *input.Title
	}
	if input.Description != nil {
		ch.Description = *input.Description
	}
	if input.Emoji != nil {
		ch.Emoji = *input.Emoji
	}
	if input.Color != nil {
		ch.Color = *input.Color
	}
	if input.EndDate != nil {
		if *input.EndDate == "" {
			ch.EndDate = nil
		} else {
			ch.EndDate = input.EndDate
		}
	}

	const q = `
		UPDATE life_chapters
		SET title = $3, description = $4, end_date = $5, emoji = $6, color = $7, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, title, description, start_date, end_date, emoji, color, summary, created_at, updated_at`

	row := r.db.QueryRow(ctx, q, id, userID, ch.Title, ch.Description, ch.EndDate, ch.Emoji, ch.Color)
	return scanChapter(row)
}

// StoreSummary persists a Claude-generated summary text.
func (r *LifeChapterRepository) StoreSummary(ctx context.Context, id, userID uuid.UUID, summary string) error {
	const q = `UPDATE life_chapters SET summary = $3, updated_at = NOW() WHERE id = $1 AND user_id = $2`
	tag, err := r.db.Exec(ctx, q, id, userID, summary)
	if err != nil {
		return fmt.Errorf("lifeChapters.StoreSummary: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("lifeChapters.StoreSummary: chapter not found")
	}
	return nil
}

// Delete removes a chapter (returns nil if not found).
func (r *LifeChapterRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	const q = `DELETE FROM life_chapters WHERE id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("lifeChapters.Delete: %w", err)
	}
	return nil
}

// GetDetail returns the chapter enriched with entry stats for the chapter's date range.
func (r *LifeChapterRepository) GetDetail(ctx context.Context, id, userID uuid.UUID) (*models.ChapterDetail, error) {
	ch, err := r.GetByID(ctx, id, userID)
	if err != nil || ch == nil {
		return nil, err
	}

	detail := &models.ChapterDetail{LifeChapter: *ch}

	// Compute the date range filter.
	endFilter := "NOW()"
	var args []any
	args = append(args, userID, ch.StartDate)
	if ch.EndDate != nil {
		endFilter = "$3"
		args = append(args, *ch.EndDate+" 23:59:59")
	}

	// Entry count + avg mood.
	statQ := fmt.Sprintf(`
		SELECT COUNT(e.id)::INT, ROUND(AVG(ea.mood_score))::INT
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND DATE(e.created_at AT TIME ZONE 'UTC') >= $2::DATE
		  AND e.created_at <= %s`, endFilter)

	var count int
	var avgMood *int
	if err := r.db.QueryRow(ctx, statQ, args...).Scan(&count, &avgMood); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lifeChapters.GetDetail stats: %w", err)
	}
	detail.EntryCount = count
	detail.AvgMood = avgMood

	// Daily mood arc.
	arcQ := fmt.Sprintf(`
		SELECT TO_CHAR(DATE(e.created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day,
		       ROUND(AVG(ea.mood_score))::INT AS avg_mood
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND DATE(e.created_at AT TIME ZONE 'UTC') >= $2::DATE
		  AND e.created_at <= %s
		GROUP BY day
		ORDER BY day ASC`, endFilter)

	arcRows, err := r.db.Query(ctx, arcQ, args...)
	if err != nil {
		return nil, fmt.Errorf("lifeChapters.GetDetail arc: %w", err)
	}
	defer arcRows.Close()
	for arcRows.Next() {
		var d models.MoodArcDay
		if err := arcRows.Scan(&d.Date, &d.AvgMood); err != nil {
			return nil, fmt.Errorf("lifeChapters.GetDetail arc scan: %w", err)
		}
		detail.MoodArc = append(detail.MoodArc, d)
	}

	// Top emotions.
	emotionQ := fmt.Sprintf(`
		SELECT em.emotion, COUNT(*)::INT AS freq
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id,
		     LATERAL jsonb_array_elements(ea.emotional_tone::jsonb) AS et,
		     LATERAL (SELECT et->>'emotion' AS emotion, (et->>'intensity')::float AS intensity) AS em
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND DATE(e.created_at AT TIME ZONE 'UTC') >= $2::DATE
		  AND e.created_at <= %s
		  AND em.intensity >= 0.4
		GROUP BY em.emotion
		ORDER BY freq DESC
		LIMIT 5`, endFilter)

	emRows, err := r.db.Query(ctx, emotionQ, args...)
	if err != nil {
		return nil, fmt.Errorf("lifeChapters.GetDetail emotions: %w", err)
	}
	defer emRows.Close()
	for emRows.Next() {
		var emotion string
		var freq int
		if err := emRows.Scan(&emotion, &freq); err != nil {
			return nil, fmt.Errorf("lifeChapters.GetDetail emotions scan: %w", err)
		}
		detail.TopEmotions = append(detail.TopEmotions, emotion)
	}

	if detail.MoodArc == nil {
		detail.MoodArc = []models.MoodArcDay{}
	}
	if detail.TopEmotions == nil {
		detail.TopEmotions = []string{}
	}

	return detail, nil
}

// GetEntriesInRange returns lightweight entry summaries that fall within the chapter.
func (r *LifeChapterRepository) GetEntriesInRange(ctx context.Context, userID uuid.UUID, startDate string, endDate *string) ([]*models.WeekSummaryEntry, error) {
	endFilter := "NOW()"
	args := []any{userID, startDate}
	if endDate != nil {
		endFilter = "$3"
		args = append(args, *endDate+" 23:59:59")
	}

	q := fmt.Sprintf(`
		SELECT DATE(e.created_at AT TIME ZONE 'UTC'), ea.summary, ea.mood_score, ea.emotional_tone
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND DATE(e.created_at AT TIME ZONE 'UTC') >= $2::DATE
		  AND e.created_at <= %s
		  AND ea.summary IS NOT NULL AND ea.summary != ''
		ORDER BY e.created_at ASC
		LIMIT 50`, endFilter)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("lifeChapters.GetEntriesInRange: %w", err)
	}
	defer rows.Close()

	var entries []*models.WeekSummaryEntry
	for rows.Next() {
		entry := &models.WeekSummaryEntry{}
		var toneRaw []byte
		if err := rows.Scan(&entry.Date, &entry.Summary, &entry.MoodScore, &toneRaw); err != nil {
			return nil, fmt.Errorf("lifeChapters.GetEntriesInRange scan: %w", err)
		}
		var tones []models.EmotionalTone
		if err2 := json.Unmarshal(toneRaw, &tones); err2 == nil {
			for _, t := range tones {
				if t.Intensity >= 0.5 {
					entry.Emotions = append(entry.Emotions, t.Emotion)
				}
			}
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

type chapterScanner interface {
	Scan(dest ...any) error
}

func scanChapter(row chapterScanner) (*models.LifeChapter, error) {
	ch := &models.LifeChapter{}
	var startDate, summary string
	var endDate *string
	var updatedAt time.Time

	err := row.Scan(
		&ch.ID, &ch.UserID, &ch.Title, &ch.Description,
		&startDate, &endDate,
		&ch.Emoji, &ch.Color, &summary,
		&ch.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	ch.StartDate = startDate
	ch.EndDate = endDate
	ch.Summary = summary
	ch.UpdatedAt = updatedAt
	return ch, nil
}

