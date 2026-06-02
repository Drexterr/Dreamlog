package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalysisRepository struct {
	db *pgxpool.Pool
}

func NewAnalysisRepository(db *pgxpool.Pool) *AnalysisRepository {
	return &AnalysisRepository{db: db}
}

// Upsert creates or fully replaces the analysis for an entry.
func (r *AnalysisRepository) Upsert(ctx context.Context, entryID uuid.UUID, a *models.EntryAnalysis) (*models.EntryAnalysis, error) {
	toneJSON, err := json.Marshal(a.EmotionalTone)
	if err != nil {
		return nil, fmt.Errorf("analysis.Upsert: marshal tone: %w", err)
	}

	const q = `
		INSERT INTO entry_analysis
		    (entry_id, mood_score, emotional_tone, topics, key_quotes, summary, reflection, morning_nudge, is_crisis, dream_symbols, dream_type)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (entry_id) DO UPDATE
		    SET mood_score    = EXCLUDED.mood_score,
		        emotional_tone = EXCLUDED.emotional_tone,
		        topics        = EXCLUDED.topics,
		        key_quotes    = EXCLUDED.key_quotes,
		        summary       = EXCLUDED.summary,
		        reflection    = EXCLUDED.reflection,
		        morning_nudge = EXCLUDED.morning_nudge,
		        is_crisis     = EXCLUDED.is_crisis,
		        dream_symbols = EXCLUDED.dream_symbols,
		        dream_type    = EXCLUDED.dream_type,
		        updated_at    = NOW()
		RETURNING id, entry_id, mood_score, emotional_tone, topics, key_quotes,
		          summary, reflection, morning_nudge, is_crisis, dream_symbols, dream_type, created_at, updated_at`

	dreamSymbols := a.DreamSymbols
	var dreamType *string
	if a.DreamType != "" {
		dreamType = &a.DreamType
	}

	row := r.db.QueryRow(ctx, q,
		entryID,
		a.MoodScore,
		toneJSON,
		a.Topics,
		a.KeyQuotes,
		a.Summary,
		a.Reflection,
		a.MorningNudge,
		a.IsCrisis,
		dreamSymbols,
		dreamType,
	)
	return scanAnalysis(row)
}

// GetByEntryID fetches the analysis for a given entry, or nil if not found.
func (r *AnalysisRepository) GetByEntryID(ctx context.Context, entryID uuid.UUID) (*models.EntryAnalysis, error) {
	const q = `
		SELECT id, entry_id, mood_score, emotional_tone, topics, key_quotes,
		       summary, reflection, morning_nudge, is_crisis, dream_symbols, dream_type, created_at, updated_at
		FROM entry_analysis
		WHERE entry_id = $1`

	row := r.db.QueryRow(ctx, q, entryID)
	a, err := scanAnalysis(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// MoodLast7Days returns daily average mood for a user over the last 7 days.
// Includes both journal entries and completed therapy sessions.
func (r *AnalysisRepository) MoodLast7Days(ctx context.Context, userID uuid.UUID) ([]*models.DailyMood, error) {
	const q = `
		SELECT
		    TO_CHAR(day, 'YYYY-MM-DD') AS day,
		    ROUND(AVG(mood))::INT      AS avg_mood,
		    COUNT(*)::INT              AS entry_count
		FROM (
		    SELECT DATE(e.created_at AT TIME ZONE 'UTC') AS day, ea.mood_score::float AS mood
		    FROM entries e
		    JOIN entry_analysis ea ON ea.entry_id = e.id
		    WHERE e.user_id = $1
		      AND e.created_at >= NOW() - INTERVAL '7 days'
		      AND ea.mood_score IS NOT NULL
		      AND ea.is_crisis = FALSE
		    UNION ALL
		    SELECT DATE(ts.ended_at AT TIME ZONE 'UTC') AS day, ts.session_mood_score::float AS mood
		    FROM therapy_sessions ts
		    WHERE ts.user_id = $1
		      AND ts.ended_at >= NOW() - INTERVAL '7 days'
		      AND ts.session_mood_score IS NOT NULL
		      AND ts.status = 'completed'
		) combined
		GROUP BY day
		ORDER BY day ASC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("analysis.MoodLast7Days: %w", err)
	}
	defer rows.Close()

	var result []*models.DailyMood
	for rows.Next() {
		dm := &models.DailyMood{}
		if err := rows.Scan(&dm.Day, &dm.AvgMood, &dm.EntryCount); err != nil {
			return nil, fmt.Errorf("analysis.MoodLast7Days scan: %w", err)
		}
		result = append(result, dm)
	}
	return result, rows.Err()
}

// StreakInfo computes the current and longest streak.
// "Active days" = distinct days with a completed entry OR a streak freeze record.
func (r *AnalysisRepository) StreakInfo(ctx context.Context, userID uuid.UUID) (*models.StreakInfo, error) {
	// Union of entry days and freeze days, newest first.
	const q = `
		SELECT DISTINCT day FROM (
			SELECT DATE(created_at AT TIME ZONE 'UTC') AS day
			FROM entries
			WHERE user_id = $1 AND status = 'completed'
			UNION
			SELECT frozen_date AS day
			FROM streak_freeze_days
			WHERE user_id = $1
		) combined
		ORDER BY day DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("analysis.StreakInfo: %w", err)
	}
	defer rows.Close()

	var days []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("analysis.StreakInfo scan: %w", err)
		}
		days = append(days, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(days) == 0 {
		return &models.StreakInfo{}, nil
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	current := 0
	longest := 0
	streak := 0
	prev := today.Add(24 * time.Hour) // sentinel

	for _, d := range days {
		d = d.UTC().Truncate(24 * time.Hour)
		diff := int(prev.Sub(d).Hours() / 24)
		if diff == 1 {
			streak++
		} else if diff > 1 {
			if streak > longest {
				longest = streak
			}
			streak = 1
		}
		prev = d
	}
	if streak > longest {
		longest = streak
	}

	// Current streak: only counts if the most recent day is today or yesterday.
	mostRecent := days[0].UTC().Truncate(24 * time.Hour)
	diffFromToday := int(today.Sub(mostRecent).Hours() / 24)
	if diffFromToday <= 1 {
		// Walk forward from newest to count current.
		current = 1
		for i := 1; i < len(days); i++ {
			diff := int(days[i-1].UTC().Truncate(24*time.Hour).Sub(days[i].UTC().Truncate(24*time.Hour)).Hours() / 24)
			if diff == 1 {
				current++
			} else {
				break
			}
		}
	}

	return &models.StreakInfo{
		CurrentStreak: current,
		LongestStreak: longest,
		TotalDays:     len(days),
		NextMilestone: models.NextStreakMilestone(current),
	}, nil
}


// MoodHistory returns daily average mood and aggregated emotion data
// for the given number of days, plus the same for the prior equal period (for delta).
func (r *AnalysisRepository) MoodHistory(ctx context.Context, userID uuid.UUID, days int) (*models.MoodHistoryResponse, error) {
	now := time.Now().UTC()
	since := now.AddDate(0, 0, -days)
	prevSince := since.AddDate(0, 0, -days)

	// Daily mood for the requested window — journal entries + therapy sessions combined.
	const dailyQ = `
		SELECT
		    TO_CHAR(day, 'YYYY-MM-DD') AS day,
		    ROUND(AVG(mood))::INT      AS avg_mood,
		    COUNT(*)::INT              AS entry_count
		FROM (
		    SELECT DATE(e.created_at AT TIME ZONE 'UTC') AS day, ea.mood_score::float AS mood
		    FROM entries e
		    JOIN entry_analysis ea ON ea.entry_id = e.id
		    WHERE e.user_id = $1
		      AND e.created_at >= $2
		      AND ea.mood_score IS NOT NULL
		      AND ea.is_crisis = FALSE
		    UNION ALL
		    SELECT DATE(ts.ended_at AT TIME ZONE 'UTC') AS day, ts.session_mood_score::float AS mood
		    FROM therapy_sessions ts
		    WHERE ts.user_id = $1
		      AND ts.ended_at >= $2
		      AND ts.session_mood_score IS NOT NULL
		      AND ts.status = 'completed'
		) combined
		GROUP BY day
		ORDER BY day ASC`

	rows, err := r.db.Query(ctx, dailyQ, userID, since)
	if err != nil {
		return nil, fmt.Errorf("analysis.MoodHistory daily: %w", err)
	}
	defer rows.Close()

	var dailyDays []*models.DailyMood
	var totalMood, totalCount int
	for rows.Next() {
		dm := &models.DailyMood{}
		if err := rows.Scan(&dm.Day, &dm.AvgMood, &dm.EntryCount); err != nil {
			return nil, fmt.Errorf("analysis.MoodHistory scan: %w", err)
		}
		dailyDays = append(dailyDays, dm)
		totalMood += dm.AvgMood * dm.EntryCount
		totalCount += dm.EntryCount
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var avgMood *int
	if totalCount > 0 {
		v := totalMood / totalCount
		avgMood = &v
	}

	// Previous period average — journal entries + therapy sessions combined.
	const prevAvgQ = `
		SELECT COALESCE(ROUND(AVG(mood))::INT, 0), COUNT(*)::INT
		FROM (
		    SELECT ea.mood_score::float AS mood
		    FROM entries e
		    JOIN entry_analysis ea ON ea.entry_id = e.id
		    WHERE e.user_id = $1
		      AND e.created_at >= $2 AND e.created_at < $3
		      AND ea.is_crisis = FALSE
		    UNION ALL
		    SELECT session_mood_score::float AS mood
		    FROM therapy_sessions
		    WHERE user_id = $1
		      AND ended_at >= $2 AND ended_at < $3
		      AND session_mood_score IS NOT NULL
		      AND status = 'completed'
		) combined`

	var prevAvgRaw, prevCount int
	if err := r.db.QueryRow(ctx, prevAvgQ, userID, prevSince, since).Scan(&prevAvgRaw, &prevCount); err != nil {
		return nil, fmt.Errorf("analysis.MoodHistory prev: %w", err)
	}

	var prevAvgMood, moodDelta *int
	if prevCount > 0 {
		prevAvgMood = &prevAvgRaw
		if avgMood != nil {
			d := *avgMood - prevAvgRaw
			moodDelta = &d
		}
	}

	// Top emotions over the period — journal entries + therapy sessions.
	const emotionQ = `
		SELECT tone FROM (
		    SELECT ea.emotional_tone AS tone
		    FROM entries e
		    JOIN entry_analysis ea ON ea.entry_id = e.id
		    WHERE e.user_id = $1
		      AND e.created_at >= $2
		      AND ea.is_crisis = FALSE
		    UNION ALL
		    SELECT session_emotional_tone AS tone
		    FROM therapy_sessions
		    WHERE user_id = $1
		      AND ended_at >= $2
		      AND session_emotional_tone IS NOT NULL
		      AND status = 'completed'
		) combined`

	eRows, err := r.db.Query(ctx, emotionQ, userID, since)
	if err != nil {
		return nil, fmt.Errorf("analysis.MoodHistory emotions: %w", err)
	}
	defer eRows.Close()

	emotionCounts := map[string]int{}
	for eRows.Next() {
		var toneRaw []byte
		if err := eRows.Scan(&toneRaw); err != nil {
			continue
		}
		var tones []models.EmotionalTone
		if err := json.Unmarshal(toneRaw, &tones); err != nil {
			continue
		}
		for _, t := range tones {
			if t.Intensity >= 0.5 {
				emotionCounts[t.Emotion]++
			}
		}
	}

	topEmotions := topN(emotionCounts, 3)

	resp := &models.MoodHistoryResponse{
		Days:        dailyDays,
		AvgMood:     avgMood,
		PrevAvgMood: prevAvgMood,
		MoodDelta:   moodDelta,
		TopEmotions: topEmotions,
		EntryCount:  totalCount,
	}
	if resp.Days == nil {
		resp.Days = []*models.DailyMood{}
	}
	return resp, nil
}

// EmotionPatterns returns the top 8 emotions with frequency and intensity data
// over the given number of days. Used for the Pattern Radar endpoint.
func (r *AnalysisRepository) EmotionPatterns(ctx context.Context, userID uuid.UUID, days int) (*models.PatternRadarResponse, error) {
	since := time.Now().UTC().AddDate(0, 0, -days)

	// Emotion aggregation across journal entries and therapy sessions.
	// We only count emotions with intensity >= 0.3 to exclude noise.
	const emotionQ = `
		SELECT
		    combined.emotion,
		    COUNT(DISTINCT combined.source_id)::INT            AS frequency,
		    ROUND(AVG(combined.intensity)::numeric, 4)::float  AS avg_intensity
		FROM (
		    SELECT e.id AS source_id,
		           (et->>'emotion')       AS emotion,
		           (et->>'intensity')::float AS intensity
		    FROM entries e
		    JOIN entry_analysis ea ON ea.entry_id = e.id,
		    LATERAL jsonb_array_elements(ea.emotional_tone::jsonb) et
		    WHERE e.user_id = $1
		      AND e.status = 'completed'
		      AND ea.is_crisis = FALSE
		      AND e.created_at >= $2
		      AND (et->>'intensity')::float >= 0.3
		    UNION ALL
		    SELECT ts.id AS source_id,
		           (et->>'emotion')       AS emotion,
		           (et->>'intensity')::float AS intensity
		    FROM therapy_sessions ts,
		    LATERAL jsonb_array_elements(ts.session_emotional_tone::jsonb) et
		    WHERE ts.user_id = $1
		      AND ts.status = 'completed'
		      AND ts.ended_at >= $2
		      AND ts.session_emotional_tone IS NOT NULL
		      AND (et->>'intensity')::float >= 0.3
		) combined
		GROUP BY combined.emotion
		ORDER BY (COUNT(DISTINCT combined.source_id)::float * AVG(combined.intensity)) DESC
		LIMIT 8`

	rows, err := r.db.Query(ctx, emotionQ, userID, since)
	if err != nil {
		return nil, fmt.Errorf("analysis.EmotionPatterns: %w", err)
	}
	defer rows.Close()

	type rawEmotion struct {
		emotion      string
		frequency    int
		avgIntensity float64
	}
	var raw []rawEmotion
	var maxScore float64
	for rows.Next() {
		var re rawEmotion
		if err := rows.Scan(&re.emotion, &re.frequency, &re.avgIntensity); err != nil {
			return nil, fmt.Errorf("analysis.EmotionPatterns scan: %w", err)
		}
		s := float64(re.frequency) * re.avgIntensity
		if s > maxScore {
			maxScore = s
		}
		raw = append(raw, re)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	patterns := make([]models.EmotionPattern, 0, len(raw))
	for _, re := range raw {
		score := 0.0
		if maxScore > 0 {
			score = math.Round((float64(re.frequency)*re.avgIntensity/maxScore)*100) / 100
		}
		patterns = append(patterns, models.EmotionPattern{
			Emotion:      re.emotion,
			Frequency:    re.frequency,
			AvgIntensity: re.avgIntensity,
			Score:        score,
		})
	}

	// Mood distribution across journal entries and therapy sessions.
	const distQ = `
		SELECT
		    COUNT(*)::INT                                                AS total,
		    COUNT(*) FILTER (WHERE mood >= 70)::INT                     AS high,
		    COUNT(*) FILTER (WHERE mood >= 40 AND mood < 70)::INT       AS neutral,
		    COUNT(*) FILTER (WHERE mood < 40)::INT                      AS low
		FROM (
		    SELECT ea.mood_score AS mood
		    FROM entries e
		    JOIN entry_analysis ea ON ea.entry_id = e.id
		    WHERE e.user_id = $1
		      AND e.status = 'completed'
		      AND ea.is_crisis = FALSE
		      AND e.created_at >= $2
		    UNION ALL
		    SELECT session_mood_score AS mood
		    FROM therapy_sessions
		    WHERE user_id = $1
		      AND status = 'completed'
		      AND ended_at >= $2
		      AND session_mood_score IS NOT NULL
		) combined`

	var total, high, neutral, low int
	if err := r.db.QueryRow(ctx, distQ, userID, since).Scan(&total, &high, &neutral, &low); err != nil {
		return nil, fmt.Errorf("analysis.EmotionPatterns dist: %w", err)
	}

	return &models.PatternRadarResponse{
		Emotions:     patterns,
		TotalEntries: total,
		MoodDistribution: models.MoodDistribution{
			High:    high,
			Neutral: neutral,
			Low:     low,
		},
	}, nil
}

// topN returns up to n keys from a count map, sorted by value descending.
func topN(counts map[string]int, n int) []string {
	type kv struct{ k string; v int }
	pairs := make([]kv, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, kv{k, v})
	}
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j].v > pairs[j-1].v; j-- {
			pairs[j], pairs[j-1] = pairs[j-1], pairs[j]
		}
	}
	if len(pairs) > n {
		pairs = pairs[:n]
	}
	result := make([]string, len(pairs))
	for i, p := range pairs {
		result[i] = p.k
	}
	return result
}

// GetWeekSummaries fetches per-entry data for the weekly review prompt.
// Returns entries completed in [since, since+7days], oldest first, excluding crisis entries.
func (r *AnalysisRepository) GetWeekSummaries(ctx context.Context, userID uuid.UUID, since time.Time) ([]*models.WeekSummaryEntry, error) {
	const q = `
		SELECT
		    DATE(e.created_at AT TIME ZONE 'UTC') AS day,
		    ea.summary,
		    ea.mood_score,
		    ea.emotional_tone
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND e.created_at >= $2
		  AND e.created_at < $3
		  AND ea.summary IS NOT NULL AND ea.summary != ''
		ORDER BY e.created_at ASC`

	until := since.Add(7 * 24 * time.Hour)
	rows, err := r.db.Query(ctx, q, userID, since, until)
	if err != nil {
		return nil, fmt.Errorf("analysis.GetWeekSummaries: %w", err)
	}
	defer rows.Close()

	var result []*models.WeekSummaryEntry
	for rows.Next() {
		entry := &models.WeekSummaryEntry{}
		var toneRaw []byte
		var tones []models.EmotionalTone

		if err := rows.Scan(&entry.Date, &entry.Summary, &entry.MoodScore, &toneRaw); err != nil {
			return nil, fmt.Errorf("analysis.GetWeekSummaries scan: %w", err)
		}
		if err := json.Unmarshal(toneRaw, &tones); err == nil {
			for _, t := range tones {
				if t.Intensity >= 0.5 {
					entry.Emotions = append(entry.Emotions, t.Emotion)
				}
			}
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

// GetYearSummaries fetches per-entry data for the annual review prompt.
// Returns entries completed in the calendar year [yearStart, yearEnd), oldest first,
// excluding crisis entries.
func (r *AnalysisRepository) GetYearSummaries(ctx context.Context, userID uuid.UUID, yearStart, yearEnd time.Time) ([]*models.YearSummaryEntry, error) {
	const q = `
		SELECT
		    e.created_at,
		    ea.summary,
		    ea.mood_score,
		    ea.emotional_tone,
		    ea.topics
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND e.created_at >= $2
		  AND e.created_at < $3
		  AND ea.summary IS NOT NULL AND ea.summary != ''
		ORDER BY e.created_at ASC`

	rows, err := r.db.Query(ctx, q, userID, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("analysis.GetYearSummaries: %w", err)
	}
	defer rows.Close()

	var result []*models.YearSummaryEntry
	for rows.Next() {
		entry := &models.YearSummaryEntry{}
		var toneRaw []byte

		if err := rows.Scan(&entry.Date, &entry.Summary, &entry.MoodScore, &toneRaw, &entry.Topics); err != nil {
			return nil, fmt.Errorf("analysis.GetYearSummaries scan: %w", err)
		}
		var tones []models.EmotionalTone
		if err := json.Unmarshal(toneRaw, &tones); err == nil {
			for _, t := range tones {
				if t.Intensity >= 0.4 {
					entry.Emotions = append(entry.Emotions, t.Emotion)
				}
			}
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

// ExportData fetches everything needed to render the PDF export for a given time window.
func (r *AnalysisRepository) ExportData(ctx context.Context, userID uuid.UUID, since, until time.Time) (*models.ExportData, error) {
	// Daily moods.
	const dailyQ = `
		SELECT
		    TO_CHAR(DATE(e.created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day,
		    ROUND(AVG(ea.mood_score))::INT AS avg_mood,
		    COUNT(e.id)::INT               AS entry_count
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.created_at >= $2 AND e.created_at < $3
		  AND ea.is_crisis = FALSE
		GROUP BY DATE(e.created_at AT TIME ZONE 'UTC')
		ORDER BY day ASC`

	rows, err := r.db.Query(ctx, dailyQ, userID, since, until)
	if err != nil {
		return nil, fmt.Errorf("ExportData daily: %w", err)
	}
	defer rows.Close()

	var dailyMoods []*models.DailyMood
	var totalMood, totalCount int
	for rows.Next() {
		dm := &models.DailyMood{}
		if err := rows.Scan(&dm.Day, &dm.AvgMood, &dm.EntryCount); err != nil {
			return nil, fmt.Errorf("ExportData daily scan: %w", err)
		}
		dailyMoods = append(dailyMoods, dm)
		totalMood += dm.AvgMood * dm.EntryCount
		totalCount += dm.EntryCount
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var avgMood *int
	if totalCount > 0 {
		v := totalMood / totalCount
		avgMood = &v
	}

	// Prior period average for delta.
	prevSince := since.Add(-(until.Sub(since)))
	const prevQ = `
		SELECT COALESCE(ROUND(AVG(ea.mood_score))::INT, 0), COUNT(e.id)::INT
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.created_at >= $2 AND e.created_at < $3
		  AND ea.is_crisis = FALSE`

	var prevRaw, prevCnt int
	if err := r.db.QueryRow(ctx, prevQ, userID, prevSince, since).Scan(&prevRaw, &prevCnt); err != nil {
		return nil, fmt.Errorf("ExportData prev: %w", err)
	}
	var prevAvg, delta *int
	if prevCnt > 0 {
		prevAvg = &prevRaw
		if avgMood != nil {
			d := *avgMood - prevRaw
			delta = &d
		}
	}

	// Top emotions.
	const emotionQ = `
		SELECT ea.emotional_tone
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.created_at >= $2 AND e.created_at < $3
		  AND ea.is_crisis = FALSE`

	eRows, err := r.db.Query(ctx, emotionQ, userID, since, until)
	if err != nil {
		return nil, fmt.Errorf("ExportData emotions: %w", err)
	}
	defer eRows.Close()

	emotionCounts := map[string]int{}
	for eRows.Next() {
		var raw []byte
		if err := eRows.Scan(&raw); err != nil {
			continue
		}
		var tones []models.EmotionalTone
		if err := json.Unmarshal(raw, &tones); err != nil {
			continue
		}
		for _, t := range tones {
			if t.Intensity >= 0.5 {
				emotionCounts[t.Emotion]++
			}
		}
	}

	// Entry summaries (oldest first, no crisis).
	const entryQ = `
		SELECT
		    e.created_at,
		    ea.summary,
		    ea.mood_score,
		    ea.topics,
		    ea.key_quotes
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.created_at >= $2 AND e.created_at < $3
		  AND ea.is_crisis = FALSE
		  AND e.status = 'completed'
		ORDER BY e.created_at ASC`

	sRows, err := r.db.Query(ctx, entryQ, userID, since, until)
	if err != nil {
		return nil, fmt.Errorf("ExportData entries: %w", err)
	}
	defer sRows.Close()

	var entries []*models.ExportEntrySummary
	for sRows.Next() {
		es := &models.ExportEntrySummary{}
		var quotes []string
		if err := sRows.Scan(&es.Date, &es.Summary, &es.MoodScore, &es.Topics, &quotes); err != nil {
			return nil, fmt.Errorf("ExportData entries scan: %w", err)
		}
		if len(quotes) > 0 {
			es.KeyQuote = quotes[0]
		}
		entries = append(entries, es)
	}
	if err := sRows.Err(); err != nil {
		return nil, err
	}

	return &models.ExportData{
		Since:       since,
		Until:       until,
		AvgMood:     avgMood,
		PrevAvgMood: prevAvg,
		MoodDelta:   delta,
		EntryCount:  totalCount,
		TopEmotions: topN(emotionCounts, 5),
		DailyMoods:  dailyMoods,
		Entries:     entries,
	}, nil
}

// ── Therapy context queries ──────────────────────────────────────────────────

// MoodAvg30Days returns the average mood over the last 30 days across journal entries and therapy sessions.
func (r *AnalysisRepository) MoodAvg30Days(ctx context.Context, userID uuid.UUID) (*float64, error) {
	const q = `
		SELECT AVG(mood)::float
		FROM (
		    SELECT ea.mood_score::float AS mood
		    FROM entries e
		    JOIN entry_analysis ea ON ea.entry_id = e.id
		    WHERE e.user_id = $1
		      AND e.status = 'completed'
		      AND ea.is_crisis = FALSE
		      AND e.created_at >= NOW() - INTERVAL '30 days'
		    UNION ALL
		    SELECT session_mood_score::float AS mood
		    FROM therapy_sessions
		    WHERE user_id = $1
		      AND status = 'completed'
		      AND ended_at >= NOW() - INTERVAL '30 days'
		      AND session_mood_score IS NOT NULL
		) combined`
	var avg *float64
	if err := r.db.QueryRow(ctx, q, userID).Scan(&avg); err != nil {
		return nil, fmt.Errorf("analysis.MoodAvg30Days: %w", err)
	}
	return avg, nil
}

// RecentSummaries returns the most recent non-crisis entry summaries, oldest first.
func (r *AnalysisRepository) RecentSummaries(ctx context.Context, userID uuid.UUID, limit int) ([]string, error) {
	const q = `
		SELECT ea.summary
		FROM entries e
		JOIN entry_analysis ea ON ea.entry_id = e.id
		WHERE e.user_id = $1
		  AND e.status = 'completed'
		  AND ea.is_crisis = FALSE
		  AND ea.summary IS NOT NULL AND ea.summary != ''
		ORDER BY e.created_at DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("analysis.RecentSummaries: %w", err)
	}
	defer rows.Close()

	var summaries []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("analysis.RecentSummaries scan: %w", err)
		}
		summaries = append(summaries, s)
	}
	// Reverse so oldest-first.
	for i, j := 0, len(summaries)-1; i < j; i, j = i+1, j-1 {
		summaries[i], summaries[j] = summaries[j], summaries[i]
	}
	return summaries, rows.Err()
}

// TopEmotions returns the most frequent high-intensity emotions over the last 30 days,
// including both journal entries and therapy sessions.
func (r *AnalysisRepository) TopEmotions(ctx context.Context, userID uuid.UUID, limit int) ([]string, error) {
	since := time.Now().UTC().AddDate(0, 0, -30)
	const q = `
		SELECT tone FROM (
		    SELECT ea.emotional_tone AS tone
		    FROM entries e
		    JOIN entry_analysis ea ON ea.entry_id = e.id
		    WHERE e.user_id = $1
		      AND e.status = 'completed'
		      AND ea.is_crisis = FALSE
		      AND e.created_at >= $2
		    UNION ALL
		    SELECT session_emotional_tone AS tone
		    FROM therapy_sessions
		    WHERE user_id = $1
		      AND status = 'completed'
		      AND ended_at >= $2
		      AND session_emotional_tone IS NOT NULL
		) combined`

	rows, err := r.db.Query(ctx, q, userID, since)
	if err != nil {
		return nil, fmt.Errorf("analysis.TopEmotions: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var tones []models.EmotionalTone
		if err := json.Unmarshal(raw, &tones); err != nil {
			continue
		}
		for _, t := range tones {
			if t.Intensity >= 0.5 {
				counts[t.Emotion]++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return topN(counts, limit), nil
}

// TopTopics returns the most frequent topics over the last 30 days,
// including both journal entries and therapy sessions.
func (r *AnalysisRepository) TopTopics(ctx context.Context, userID uuid.UUID, limit int) ([]string, error) {
	since := time.Now().UTC().AddDate(0, 0, -30)
	const q = `
		SELECT topics FROM (
		    SELECT ea.topics AS topics
		    FROM entries e
		    JOIN entry_analysis ea ON ea.entry_id = e.id
		    WHERE e.user_id = $1
		      AND e.status = 'completed'
		      AND ea.is_crisis = FALSE
		      AND e.created_at >= $2
		      AND ea.topics IS NOT NULL
		    UNION ALL
		    SELECT session_topics AS topics
		    FROM therapy_sessions
		    WHERE user_id = $1
		      AND status = 'completed'
		      AND ended_at >= $2
		      AND session_topics IS NOT NULL
		) combined`

	rows, err := r.db.Query(ctx, q, userID, since)
	if err != nil {
		return nil, fmt.Errorf("analysis.TopTopics: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var topics []string
		if err := rows.Scan(&topics); err != nil {
			continue
		}
		for _, t := range topics {
			counts[t]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return topN(counts, limit), nil
}

func scanAnalysis(row pgx.Row) (*models.EntryAnalysis, error) {
	a := &models.EntryAnalysis{}
	var toneRaw []byte
	var dreamType *string

	err := row.Scan(
		&a.ID, &a.EntryID, &a.MoodScore, &toneRaw,
		&a.Topics, &a.KeyQuotes, &a.Summary, &a.Reflection,
		&a.MorningNudge, &a.IsCrisis, &a.DreamSymbols, &dreamType,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanAnalysis: %w", err)
	}

	if err := json.Unmarshal(toneRaw, &a.EmotionalTone); err != nil {
		return nil, fmt.Errorf("scanAnalysis: unmarshal tone: %w", err)
	}
	if dreamType != nil {
		a.DreamType = *dreamType
	}
	return a, nil
}
