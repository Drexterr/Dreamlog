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

// GetMap returns all people for a user, sorted by most recently mentioned.
func (r *RelationshipRepository) GetMap(ctx context.Context, userID uuid.UUID) ([]*models.Person, error) {
	const q = `
		SELECT id, user_id, name, role, mention_count, positive_count, negative_count,
		       last_mentioned_at, created_at, updated_at
		FROM people
		WHERE user_id = $1
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
		       last_mentioned_at, created_at, updated_at
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

// ── Helpers ───────────────────────────────────────────────────────────────────

type personScanner interface {
	Scan(dest ...any) error
}

func scanPerson(row personScanner) (*models.Person, error) {
	p := &models.Person{}
	err := row.Scan(
		&p.ID, &p.UserID, &p.Name, &p.Role,
		&p.MentionCount, &p.PositiveCount, &p.NegativeCount,
		&p.LastMentionedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}
