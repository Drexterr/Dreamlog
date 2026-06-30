package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RelationshipRepository struct {
	db *pgxpool.Pool
}

func NewRelationshipRepository(db *pgxpool.Pool) *RelationshipRepository {
	return &RelationshipRepository{db: db}
}

// UpsertPersonMentions upserts each extracted person and records a mention
// for this entry. Idempotent: calling twice for the same entry is safe.
func (r *RelationshipRepository) UpsertPersonMentions(
	ctx context.Context,
	userID, entryID uuid.UUID,
	people []models.ExtractedPerson,
) error {
	if len(people) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("relationships.UpsertPersonMentions: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, ep := range people {
		if ep.Name == "" {
			continue
		}
		role := ep.Role
		if role == "" {
			role = "other"
		}
		sentiment := ep.Sentiment
		if sentiment == "" {
			sentiment = "neutral"
		}

		posInc, negInc := 0, 0
		if sentiment == "positive" {
			posInc = 1
		} else if sentiment == "negative" {
			negInc = 1
		}

		// Upsert person (case-insensitive name match via the unique index).
		const upsertPerson = `
			INSERT INTO people (user_id, name, role, mention_count, positive_count, negative_count, last_mentioned_at)
			VALUES ($1, $2, $3, 1, $4, $5, NOW())
			ON CONFLICT (user_id, LOWER(name)) DO UPDATE
			    SET mention_count     = people.mention_count + 1,
			        positive_count    = people.positive_count + $4,
			        negative_count    = people.negative_count + $5,
			        last_mentioned_at = NOW(),
			        updated_at        = NOW()
			RETURNING id`

		var personID uuid.UUID
		if err := tx.QueryRow(ctx, upsertPerson, userID, ep.Name, role, posInc, negInc).Scan(&personID); err != nil {
			return fmt.Errorf("relationships.UpsertPersonMentions: upsert person %q: %w", ep.Name, err)
		}

		// Insert mention (ignore if already exists for this entry).
		const insertMention = `
			INSERT INTO person_mentions (person_id, entry_id, user_id, sentiment, context)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (person_id, entry_id) DO NOTHING`

		if _, err := tx.Exec(ctx, insertMention, personID, entryID, userID, sentiment, ep.Context); err != nil {
			return fmt.Errorf("relationships.UpsertPersonMentions: insert mention %q: %w", ep.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("relationships.UpsertPersonMentions: commit: %w", err)
	}
	return nil
}

// GetMap returns all non-hidden people for a user, sorted by most recently mentioned.
func (r *RelationshipRepository) GetMap(ctx context.Context, userID uuid.UUID) ([]*models.Person, error) {
	const q = `
		SELECT id, user_id, name, role, mention_count, positive_count, negative_count,
		       last_mentioned_at, hidden, created_at, updated_at
		FROM people
		WHERE user_id = $1 AND hidden = false
		ORDER BY last_mentioned_at DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("relationships.GetMap: %w", err)
	}
	defer rows.Close()

	var people []*models.Person
	for rows.Next() {
		p, err := scanPerson(rows)
		if err != nil {
			return nil, fmt.Errorf("relationships.GetMap scan: %w", err)
		}
		people = append(people, p)
	}
	return people, rows.Err()
}

// GetDetail returns a single person and their most recent 20 mentions.
func (r *RelationshipRepository) GetDetail(ctx context.Context, personID, userID uuid.UUID) (*models.PersonDetail, error) {
	const personQ = `
		SELECT id, user_id, name, role, mention_count, positive_count, negative_count,
		       last_mentioned_at, hidden, created_at, updated_at
		FROM people
		WHERE id = $1 AND user_id = $2`

	person, err := scanPerson(r.db.QueryRow(ctx, personQ, personID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("relationships.GetDetail person: %w", err)
	}

	const mentionsQ = `
		SELECT id, person_id, entry_id, user_id, sentiment, context, created_at
		FROM person_mentions
		WHERE person_id = $1
		ORDER BY created_at DESC
		LIMIT 20`

	mRows, err := r.db.Query(ctx, mentionsQ, personID)
	if err != nil {
		return nil, fmt.Errorf("relationships.GetDetail mentions: %w", err)
	}
	defer mRows.Close()

	var mentions []models.PersonMention
	for mRows.Next() {
		var m models.PersonMention
		if err := mRows.Scan(
			&m.ID, &m.PersonID, &m.EntryID, &m.UserID,
			&m.Sentiment, &m.Context, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("relationships.GetDetail mentions scan: %w", err)
		}
		mentions = append(mentions, m)
	}
	if err := mRows.Err(); err != nil {
		return nil, fmt.Errorf("relationships.GetDetail mentions rows: %w", err)
	}
	if mentions == nil {
		mentions = []models.PersonMention{}
	}

	return &models.PersonDetail{Person: person, Mentions: mentions}, nil
}

// UpdatePerson applies optional edits (name, role, hidden). Nil fields are
// unchanged. Returns nil if the person doesn't exist for this user. A rename
// that collides with an existing person (case-insensitive) returns a unique
// violation error, which the handler maps to 409.
func (r *RelationshipRepository) UpdatePerson(
	ctx context.Context,
	personID, userID uuid.UUID,
	input models.UpdatePersonInput,
) (*models.Person, error) {
	const q = `
		UPDATE people SET
		    name       = COALESCE($3, name),
		    role       = COALESCE($4, role),
		    hidden     = COALESCE($5, hidden),
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, role, mention_count, positive_count, negative_count,
		          last_mentioned_at, hidden, created_at, updated_at`

	person, err := scanPerson(r.db.QueryRow(ctx, q, personID, userID, input.Name, input.Role, input.Hidden))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("relationships.UpdatePerson: %w", err)
	}
	return person, nil
}

// MergePeople folds sourceID into targetID: all of source's mentions are
// reassigned to target, target's aggregate counts are recomputed, and source
// is deleted. Both must belong to userID. Returns nil if either is missing.
func (r *RelationshipRepository) MergePeople(
	ctx context.Context,
	targetID, sourceID, userID uuid.UUID,
) (*models.Person, error) {
	if targetID == sourceID {
		return nil, fmt.Errorf("relationships.MergePeople: cannot merge a person into itself")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("relationships.MergePeople: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Both people must exist and belong to the user.
	var owned int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM people WHERE id = ANY($1) AND user_id = $2`,
		[]uuid.UUID{targetID, sourceID}, userID,
	).Scan(&owned); err != nil {
		return nil, fmt.Errorf("relationships.MergePeople: ownership check: %w", err)
	}
	if owned != 2 {
		return nil, nil // one (or both) not found for this user
	}

	// Drop source mentions that would collide with target on the same entry
	// (the (person_id, entry_id) unique constraint), then reassign the rest.
	if _, err := tx.Exec(ctx, `
		DELETE FROM person_mentions sm
		WHERE sm.person_id = $1
		  AND EXISTS (
		      SELECT 1 FROM person_mentions tm
		      WHERE tm.person_id = $2 AND tm.entry_id = sm.entry_id
		  )`, sourceID, targetID); err != nil {
		return nil, fmt.Errorf("relationships.MergePeople: dedupe mentions: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE person_mentions SET person_id = $1 WHERE person_id = $2`,
		targetID, sourceID); err != nil {
		return nil, fmt.Errorf("relationships.MergePeople: reassign mentions: %w", err)
	}

	// Recompute target aggregates from its mentions.
	if _, err := tx.Exec(ctx, `
		UPDATE people p SET
		    mention_count     = sub.cnt,
		    positive_count    = sub.pos,
		    negative_count    = sub.neg,
		    last_mentioned_at = COALESCE(sub.last, p.last_mentioned_at),
		    updated_at        = NOW()
		FROM (
		    SELECT COUNT(*)                                          AS cnt,
		           COUNT(*) FILTER (WHERE sentiment = 'positive')   AS pos,
		           COUNT(*) FILTER (WHERE sentiment = 'negative')   AS neg,
		           MAX(created_at)                                  AS last
		    FROM person_mentions WHERE person_id = $1
		) sub
		WHERE p.id = $1`, targetID); err != nil {
		return nil, fmt.Errorf("relationships.MergePeople: recompute aggregates: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM people WHERE id = $1 AND user_id = $2`, sourceID, userID); err != nil {
		return nil, fmt.Errorf("relationships.MergePeople: delete source: %w", err)
	}

	target, err := scanPerson(tx.QueryRow(ctx, `
		SELECT id, user_id, name, role, mention_count, positive_count, negative_count,
		       last_mentioned_at, hidden, created_at, updated_at
		FROM people WHERE id = $1 AND user_id = $2`, targetID, userID))
	if err != nil {
		return nil, fmt.Errorf("relationships.MergePeople: load target: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("relationships.MergePeople: commit: %w", err)
	}
	return target, nil
}

// TopPeople returns the most-mentioned (non-hidden) people for a user as short
// strings combining name + sentiment lean, e.g. "Mom — mostly warm". Used to
// give Therapy Mode relational awareness at session start.
func (r *RelationshipRepository) TopPeople(ctx context.Context, userID uuid.UUID, limit int) ([]string, error) {
	const q = `
		SELECT name, positive_count, negative_count, mention_count
		FROM people
		WHERE user_id = $1 AND hidden = false AND mention_count >= 2
		ORDER BY mention_count DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("relationships.TopPeople: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		var pos, neg, total int
		if err := rows.Scan(&name, &pos, &neg, &total); err != nil {
			return nil, fmt.Errorf("relationships.TopPeople scan: %w", err)
		}
		out = append(out, fmt.Sprintf("%s — %s", name, sentimentLean(pos, neg, total)))
	}
	return out, rows.Err()
}

// sentimentLean summarizes the emotional valence around a person in plain words.
func sentimentLean(pos, neg, total int) string {
	switch {
	case pos > neg*2 && pos > 0:
		return "mostly warm"
	case neg > pos*2 && neg > 0:
		return "often difficult"
	case pos > 0 || neg > 0:
		return "mixed feelings"
	default:
		return "mostly neutral"
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

type personScanner interface {
	Scan(dest ...any) error
}

func scanPerson(row personScanner) (*models.Person, error) {
	p := &models.Person{}
	err := row.Scan(
		&p.ID, &p.UserID, &p.Name, &p.Role,
		&p.MentionCount, &p.PositiveCount, &p.NegativeCount,
		&p.LastMentionedAt, &p.Hidden, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}
