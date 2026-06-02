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

type TherapistRepository struct {
	db *pgxpool.Pool
}

func NewTherapistRepository(db *pgxpool.Pool) *TherapistRepository {
	return &TherapistRepository{db: db}
}

// GetByUserID returns the therapist profile for an authenticated user, or nil.
func (r *TherapistRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.Therapist, error) {
	const q = `
		SELECT id, user_id, name, email, credentials, plan, created_at, updated_at
		FROM therapists WHERE user_id = $1`
	t := &models.Therapist{}
	err := r.db.QueryRow(ctx, q, userID).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Email, &t.Credentials, &t.Plan, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("therapist.GetByUserID: %w", err)
	}
	return t, nil
}

// Register inserts a new therapist record (idempotent on user_id conflict).
func (r *TherapistRepository) Register(ctx context.Context, userID uuid.UUID, name, email, credentials string) (*models.Therapist, error) {
	const q = `
		INSERT INTO therapists (user_id, name, email, credentials)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		    SET name = EXCLUDED.name, email = EXCLUDED.email,
		        credentials = EXCLUDED.credentials, updated_at = NOW()
		RETURNING id, user_id, name, email, credentials, plan, created_at, updated_at`

	t := &models.Therapist{}
	err := r.db.QueryRow(ctx, q, userID, name, email, credentials).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Email, &t.Credentials, &t.Plan, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("therapist.Register: %w", err)
	}
	return t, nil
}

// LinkClient creates an active link between therapist and client (idempotent).
func (r *TherapistRepository) LinkClient(ctx context.Context, therapistID, clientID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO client_therapist_links (therapist_id, client_id)
		VALUES ($1, $2)
		ON CONFLICT (therapist_id, client_id) DO UPDATE SET status = 'active', revoked_at = NULL`,
		therapistID, clientID,
	)
	if err != nil {
		return fmt.Errorf("therapist.LinkClient: %w", err)
	}
	return nil
}

// UnlinkClient soft-revokes the link.
func (r *TherapistRepository) UnlinkClient(ctx context.Context, therapistID, clientID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE client_therapist_links
		SET status = 'revoked', revoked_at = NOW()
		WHERE therapist_id = $1 AND client_id = $2`,
		therapistID, clientID,
	)
	if err != nil {
		return fmt.Errorf("therapist.UnlinkClient: %w", err)
	}
	return nil
}

// ListClients returns all active clients with lightweight stats.
func (r *TherapistRepository) ListClients(ctx context.Context, therapistID uuid.UUID) ([]*models.ClientSummary, error) {
	const q = `
		SELECT
		    u.id,
		    COALESCE(u.preferred_name, u.name) AS display_name,
		    ctl.linked_at,
		    MAX(e.created_at)                   AS last_entry_at,
		    ROUND(AVG(ea.mood_score))::INT       AS avg_mood_30d,
		    COUNT(e.id)::INT                     AS entry_count
		FROM client_therapist_links ctl
		JOIN users u ON u.id = ctl.client_id
		LEFT JOIN entries e ON e.user_id = u.id
		    AND e.status = 'completed'
		    AND e.created_at >= NOW() - INTERVAL '30 days'
		LEFT JOIN entry_analysis ea ON ea.entry_id = e.id AND ea.is_crisis = FALSE
		WHERE ctl.therapist_id = $1 AND ctl.status = 'active'
		GROUP BY u.id, u.preferred_name, u.name, ctl.linked_at
		ORDER BY last_entry_at DESC NULLS LAST`

	rows, err := r.db.Query(ctx, q, therapistID)
	if err != nil {
		return nil, fmt.Errorf("therapist.ListClients: %w", err)
	}
	defer rows.Close()

	var clients []*models.ClientSummary
	for rows.Next() {
		cs := &models.ClientSummary{}
		if err := rows.Scan(
			&cs.ClientID, &cs.Name, &cs.LinkedAt, &cs.LastEntryAt,
			&cs.AvgMood30d, &cs.EntryCount,
		); err != nil {
			return nil, fmt.Errorf("therapist.ListClients scan: %w", err)
		}
		clients = append(clients, cs)
	}
	return clients, rows.Err()
}

// GetClientLink confirms there is an active link between therapist and client.
func (r *TherapistRepository) GetClientLink(ctx context.Context, therapistID, clientID uuid.UUID) (*models.ClientTherapistLink, error) {
	const q = `
		SELECT id, therapist_id, client_id, status, linked_at, revoked_at
		FROM client_therapist_links
		WHERE therapist_id = $1 AND client_id = $2 AND status = 'active'`

	l := &models.ClientTherapistLink{}
	err := r.db.QueryRow(ctx, q, therapistID, clientID).Scan(
		&l.ID, &l.TherapistID, &l.ClientID, &l.Status, &l.LinkedAt, &l.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("therapist.GetClientLink: %w", err)
	}
	return l, nil
}

// ClientRecentEntries returns the last 5 completed, non-crisis entries for a client.
func (r *TherapistRepository) ClientRecentEntries(ctx context.Context, clientID uuid.UUID) ([]*models.ExportEntrySummary, error) {
	const q = `
		SELECT e.created_at, ea.summary, ea.mood_score, ea.topics, ea.key_quotes
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1 AND e.status = 'completed' AND ea.is_crisis = FALSE
		ORDER BY e.created_at DESC
		LIMIT 5`

	rows, err := r.db.Query(ctx, q, clientID)
	if err != nil {
		return nil, fmt.Errorf("therapist.ClientRecentEntries: %w", err)
	}
	defer rows.Close()

	var entries []*models.ExportEntrySummary
	for rows.Next() {
		es := &models.ExportEntrySummary{}
		var quotes []string
		if err := rows.Scan(&es.Date, &es.Summary, &es.MoodScore, &es.Topics, &quotes); err != nil {
			return nil, fmt.Errorf("therapist.ClientRecentEntries scan: %w", err)
		}
		if len(quotes) > 0 {
			es.KeyQuote = quotes[0]
		}
		entries = append(entries, es)
	}
	return entries, rows.Err()
}

// ClientMoodStats returns avg_mood and top emotions over the last 7 and 30 days.
func (r *TherapistRepository) ClientMoodStats(ctx context.Context, clientID uuid.UUID) (avg7d *int, topEmotions []string, trend string, err error) {
	const moodQ = `
		SELECT ROUND(AVG(ea.mood_score))::INT, COUNT(e.id)::INT
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1 AND e.status = 'completed' AND ea.is_crisis = FALSE
		  AND e.created_at >= NOW() - INTERVAL '7 days'`

	var avg7dRaw, cnt7d int
	_ = r.db.QueryRow(ctx, moodQ, clientID).Scan(&avg7dRaw, &cnt7d)
	if cnt7d > 0 {
		avg7d = &avg7dRaw
	}

	// Prior 7 days for trend.
	const prevQ = `
		SELECT ROUND(AVG(ea.mood_score))::INT, COUNT(e.id)::INT
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1 AND e.status = 'completed' AND ea.is_crisis = FALSE
		  AND e.created_at >= NOW() - INTERVAL '14 days'
		  AND e.created_at <  NOW() - INTERVAL '7 days'`

	var prevAvg, prevCnt int
	_ = r.db.QueryRow(ctx, prevQ, clientID).Scan(&prevAvg, &prevCnt)

	trend = "insufficient_data"
	if cnt7d > 0 && prevCnt > 0 {
		delta := avg7dRaw - prevAvg
		switch {
		case delta >= 5:
			trend = "improving"
		case delta <= -5:
			trend = "declining"
		default:
			trend = "stable"
		}
	}

	// Top emotions last 30d.
	const emotionQ = `
		SELECT ea.emotional_tone
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1 AND e.status = 'completed' AND ea.is_crisis = FALSE
		  AND e.created_at >= NOW() - INTERVAL '30 days'`

	eRows, qErr := r.db.Query(ctx, emotionQ, clientID)
	if qErr != nil {
		return avg7d, nil, trend, nil
	}
	defer eRows.Close()

	counts := map[string]int{}
	for eRows.Next() {
		var raw []byte
		if sErr := eRows.Scan(&raw); sErr != nil {
			continue
		}
		var tones []models.EmotionalTone
		if uErr := json.Unmarshal(raw, &tones); uErr != nil {
			continue
		}
		for _, t := range tones {
			if t.Intensity >= 0.5 {
				counts[t.Emotion]++
			}
		}
	}
	topEmotions = topN(counts, 5)

	return avg7d, topEmotions, trend, nil
}

// ClientEntryCount returns total completed non-crisis entry count for a user.
func (r *TherapistRepository) ClientEntryCount(ctx context.Context, clientID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(e.id) FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1 AND e.status = 'completed' AND ea.is_crisis = FALSE`,
		clientID,
	).Scan(&n)
	return n, err
}

// ClientDisplayName returns the display name for a user.
func (r *TherapistRepository) ClientDisplayName(ctx context.Context, clientID uuid.UUID) (string, error) {
	var name string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(preferred_name, name) FROM users WHERE id = $1`, clientID,
	).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}

// ClientRecentSummariesText returns summaries as a single text block for Claude.
func (r *TherapistRepository) ClientRecentSummariesText(ctx context.Context, clientID uuid.UUID, since time.Time) (string, error) {
	const q = `
		SELECT TO_CHAR(e.created_at, 'Mon DD'), ea.summary, ea.mood_score
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1 AND e.status = 'completed' AND ea.is_crisis = FALSE
		  AND e.created_at >= $2
		ORDER BY e.created_at DESC
		LIMIT 7`

	rows, err := r.db.Query(ctx, q, clientID, since)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var out string
	for rows.Next() {
		var dateStr, summary string
		var score int
		if err := rows.Scan(&dateStr, &summary, &score); err != nil {
			continue
		}
		out += fmt.Sprintf("[%s | mood %d] %s\n", dateStr, score, summary)
	}
	return out, rows.Err()
}
