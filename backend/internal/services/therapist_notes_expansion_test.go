package services

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	pkgcrypto "github.com/dreamlog/backend/pkg/crypto"
	"github.com/google/uuid"
)

var notesExpansionFailure = errors.New("notes expansion injected failure")

type notesExpansionRepo struct {
	*fakeNotesRepo
	fail     string
	limit    int
	listKind string
	deletes  int
}

func (r *notesExpansionRepo) GetWrappedDEK(ctx context.Context, id uuid.UUID) ([]byte, error) {
	if r.fail == "key" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.GetWrappedDEK(ctx, id)
}
func (r *notesExpansionRepo) InsertWrappedDEK(ctx context.Context, id uuid.UUID, b []byte) ([]byte, error) {
	if r.fail == "insert-key" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.InsertWrappedDEK(ctx, id, b)
}
func (r *notesExpansionRepo) CreateExternalClient(ctx context.Context, clientID, id uuid.UUID, b []byte, role string) (*models.ExternalClientRow, error) {
	if r.fail == "create-client" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.CreateExternalClient(ctx, clientID, id, b, role)
}
func (r *notesExpansionRepo) GetExternalClient(ctx context.Context, id, client uuid.UUID) (*models.ExternalClientRow, error) {
	if r.fail == "get-client" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.GetExternalClient(ctx, id, client)
}
func (r *notesExpansionRepo) ListExternalClients(ctx context.Context, id uuid.UUID, archived bool) ([]*models.ExternalClientRow, error) {
	if r.fail == "list-clients" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.ListExternalClients(ctx, id, archived)
}
func (r *notesExpansionRepo) UpdateExternalClient(ctx context.Context, id, client uuid.UUID, b []byte, role *string, archived *bool) (bool, error) {
	if r.fail == "update-client" {
		return false, notesExpansionFailure
	}
	return r.fakeNotesRepo.UpdateExternalClient(ctx, id, client, b, role, archived)
}
func (r *notesExpansionRepo) DeleteExternalClient(ctx context.Context, id, client uuid.UUID) (bool, error) {
	if r.fail == "delete-client" {
		return false, notesExpansionFailure
	}
	return r.fakeNotesRepo.DeleteExternalClient(ctx, id, client)
}
func (r *notesExpansionRepo) CreateSession(ctx context.Context, row *models.ClientSessionRow) (*models.ClientSessionRow, error) {
	if r.fail == "create-session" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.CreateSession(ctx, row)
}
func (r *notesExpansionRepo) GetSession(ctx context.Context, id, session uuid.UUID) (*models.ClientSessionRow, error) {
	if r.fail == "get-session" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.GetSession(ctx, id, session)
}
func (r *notesExpansionRepo) ListRecentSessions(ctx context.Context, id uuid.UUID, limit int) ([]*models.ClientSessionRow, error) {
	r.limit, r.listKind = limit, "recent"
	if r.fail == "list-sessions" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.ListRecentSessions(ctx, id, limit)
}
func (r *notesExpansionRepo) ListSessionsForExternalClient(ctx context.Context, id, client uuid.UUID, limit int) ([]*models.ClientSessionRow, error) {
	r.limit, r.listKind = limit, "external"
	if r.fail == "list-sessions" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.ListSessionsForExternalClient(ctx, id, client, limit)
}
func (r *notesExpansionRepo) ListSessionsForLinkedClient(ctx context.Context, id, client uuid.UUID, limit int) ([]*models.ClientSessionRow, error) {
	r.limit, r.listKind = limit, "linked"
	if r.fail == "list-sessions" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.ListSessionsForLinkedClient(ctx, id, client, limit)
}
func (r *notesExpansionRepo) UpdateSessionBullets(ctx context.Context, id, session uuid.UUID, b []byte) (bool, error) {
	if r.fail == "update-bullets" {
		return false, notesExpansionFailure
	}
	return r.fakeNotesRepo.UpdateSessionBullets(ctx, id, session, b)
}
func (r *notesExpansionRepo) UpdateSessionSummary(ctx context.Context, id, session uuid.UUID, b []byte) (bool, error) {
	if r.fail == "update-summary" {
		return false, notesExpansionFailure
	}
	return r.fakeNotesRepo.UpdateSessionSummary(ctx, id, session, b)
}
func (r *notesExpansionRepo) DeleteSession(ctx context.Context, id, session uuid.UUID) (*string, bool, error) {
	r.deletes++
	if r.fail == "delete-session" {
		return nil, false, notesExpansionFailure
	}
	return r.fakeNotesRepo.DeleteSession(ctx, id, session)
}
func (r *notesExpansionRepo) Overview(ctx context.Context, id uuid.UUID) (*models.TherapistOverview, error) {
	if r.fail == "overview" {
		return nil, notesExpansionFailure
	}
	return r.fakeNotesRepo.Overview(ctx, id)
}

type notesExpansionStorage struct {
	fail    bool
	put     string
	deleted []string
}

func (s *notesExpansionStorage) PresignPutKey(_ context.Context, key string) (string, error) {
	s.put = key
	if s.fail {
		return "", notesExpansionFailure
	}
	return "https://notes.example/put", nil
}
func (s *notesExpansionStorage) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	if s.fail {
		return notesExpansionFailure
	}
	return nil
}

type notesExpansionQueue struct {
	fail bool
	jobs []*models.NoteOCRJob
}

func (q *notesExpansionQueue) Enqueue(_ context.Context, v any) error {
	q.jobs = append(q.jobs, v.(*models.NoteOCRJob))
	if q.fail {
		return notesExpansionFailure
	}
	return nil
}

type notesExpansionLinks struct{ fail bool }

func (l notesExpansionLinks) GetClientLink(_ context.Context, _, _ uuid.UUID) (*models.ClientTherapistLink, error) {
	if l.fail {
		return nil, notesExpansionFailure
	}
	return nil, nil
}

type notesExpansionSummary struct{ run func() }

func (s notesExpansionSummary) GenerateSessionNotesSummary(_ context.Context, _, _ string, _ []string) (string, error) {
	if s.run == nil {
		return "", notesExpansionFailure
	}
	s.run()
	return "private summary", nil
}

func newNotesExpansionService(t *testing.T) (*TherapistNotesService, *notesExpansionRepo, *notesExpansionStorage, *notesExpansionQueue, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	r := &notesExpansionRepo{fakeNotesRepo: newFakeNotesRepo()}
	storage, queue := &notesExpansionStorage{}, &notesExpansionQueue{}
	master, err := pkgcrypto.NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	s := NewTherapistNotesService(r, notesExpansionLinks{}, queue, storage, &fakeSummarizer{summary: "summary"}, NewNotesCipher(master, r))
	id := uuid.New()
	client, err := s.CreateExternalClient(context.Background(), id, "  Private Client  ", "")
	if err != nil {
		t.Fatal(err)
	}
	session, err := s.CreateSession(context.Background(), id, CreateSessionInput{ExternalClientID: &client.ID, SessionDate: "2026-10-14", Bullets: []string{" original "}})
	if err != nil {
		t.Fatal(err)
	}
	return s, r, storage, queue, id, client.ID, session.ID
}

func TestNotesExpansionRepositoryFailures(t *testing.T) {
	for _, op := range []string{"create-client", "get-client", "list-clients", "update-client", "delete-client", "create-session", "get-session", "list-sessions", "update-bullets", "update-summary", "delete-session", "overview"} {
		t.Run(op, func(t *testing.T) {
			s, r, _, q, id, client, session := newNotesExpansionService(t)
			r.fail = op
			ctx := context.Background()
			var err error
			switch op {
			case "create-client":
				_, err = s.CreateExternalClient(ctx, id, "Name", "client")
			case "get-client":
				_, err = s.GetExternalClient(ctx, id, client)
				if !errors.Is(err, notesExpansionFailure) {
					t.Fatal(err)
				}
				_, err = s.CreateSession(ctx, id, CreateSessionInput{ExternalClientID: &client, Bullets: []string{"note"}})
			case "list-clients":
				_, err = s.ListExternalClients(ctx, id, true)
			case "update-client":
				name := "New"
				_, err = s.UpdateExternalClient(ctx, id, client, UpdateExternalClientInput{Name: &name})
			case "delete-client":
				err = s.DeleteExternalClient(ctx, id, client)
			case "create-session":
				_, err = s.CreateSession(ctx, id, CreateSessionInput{ExternalClientID: &client, ImageKey: "notes/" + id.String() + "/photo.jpg"})
			case "get-session":
				_, err = s.GetSession(ctx, id, session)
				if !errors.Is(err, notesExpansionFailure) {
					t.Fatal(err)
				}
				_, err = s.Summarize(ctx, id, session)
			case "list-sessions":
				_, err = s.ListSessions(ctx, id, nil, nil, 50)
			case "update-bullets":
				_, err = s.UpdateBullets(ctx, id, session, []string{"new"})
			case "update-summary":
				_, err = s.Summarize(ctx, id, session)
			case "delete-session":
				err = s.DeleteSession(ctx, id, session)
			case "overview":
				_, err = s.Overview(ctx, id)
			}
			if !errors.Is(err, notesExpansionFailure) {
				t.Fatalf("want original failure, got %v", err)
			}
			if len(q.jobs) != 0 {
				t.Fatal("queued work despite repository failure")
			}
		})
	}
}

func TestNotesExpansionClientsLifecycleAndIsolation(t *testing.T) {
	s, r, _, _, id, client, _ := newNotesExpansionService(t)
	ctx := context.Background()
	name, role, archived := "  Renamed  ", "patient", true
	got, err := s.UpdateExternalClient(ctx, id, client, UpdateExternalClientInput{Name: &name, Role: &role, Archived: &archived})
	if err != nil || got.Name != "Renamed" || got.Role != role || !got.Archived {
		t.Fatalf("update: %+v %v", got, err)
	}
	if strings.Contains(string(r.clients[client].NameEnc), "Renamed") {
		t.Fatal("plaintext name persisted")
	}
	for _, include := range []bool{false, true} {
		list, err := s.ListExternalClients(ctx, id, include)
		want := 0
		if include {
			want = 1
		}
		if err != nil || len(list) != want {
			t.Fatalf("archived=%v: %+v %v", include, list, err)
		}
	}
	other := uuid.New()
	_, err = s.GetExternalClient(ctx, other, client)
	isAPIErrorWithCode(t, err, 404)
	_, err = s.UpdateExternalClient(ctx, other, client, UpdateExternalClientInput{Role: &role})
	isAPIErrorWithCode(t, err, 404)
	isAPIErrorWithCode(t, s.DeleteExternalClient(ctx, other, client), 404)
	if err := s.DeleteExternalClient(ctx, id, client); err != nil {
		t.Fatal(err)
	}
	isAPIErrorWithCode(t, s.DeleteExternalClient(ctx, id, client), 404)
	_, err = s.UpdateExternalClient(ctx, id, client, UpdateExternalClientInput{})
	isAPIErrorWithCode(t, err, 400)
	name = "  "
	_, err = s.UpdateExternalClient(ctx, id, client, UpdateExternalClientInput{Name: &name})
	isAPIErrorWithCode(t, err, 400)
	_, err = s.CreateExternalClient(ctx, id, strings.Repeat("x", 201), "")
	isAPIErrorWithCode(t, err, 400)
}

func TestNotesExpansionSessionsListsUpdatesAndOwnership(t *testing.T) {
	s, r, _, _, id, client, session := newNotesExpansionService(t)
	ctx := context.Background()
	linked := uuid.New()
	s.links = &fakeLinkChecker{activeLinks: map[uuid.UUID]bool{linked: true}}
	_, err := s.CreateSession(ctx, id, CreateSessionInput{LinkedClientID: &linked, Bullets: []string{"linked"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		kind                    string
		ext, linked             *uuid.UUID
		limit, wantLimit, count int
	}{
		{"external", &client, nil, -1, 50, 1}, {"linked", nil, &linked, 101, 50, 1}, {"recent", nil, nil, 7, 7, 2},
	} {
		list, err := s.ListSessions(ctx, id, tc.ext, tc.linked, tc.limit)
		if err != nil || len(list) != tc.count || r.limit != tc.wantLimit || r.listKind != tc.kind {
			t.Fatalf("%s: %+v %v limit=%d kind=%s", tc.kind, list, err, r.limit, r.listKind)
		}
	}
	got, err := s.UpdateBullets(ctx, id, session, []string{"  revised ", " ", "second"})
	if err != nil || !reflect.DeepEqual(got.Bullets, []string{"revised", "second"}) {
		t.Fatalf("update: %+v %v", got, err)
	}
	other := uuid.New()
	_, err = s.GetSession(ctx, other, session)
	isAPIErrorWithCode(t, err, 404)
	_, err = s.UpdateBullets(ctx, other, session, []string{"intruder"})
	isAPIErrorWithCode(t, err, 404)
	_, err = s.Summarize(ctx, other, session)
	isAPIErrorWithCode(t, err, 404)
	isAPIErrorWithCode(t, s.DeleteSession(ctx, other, session), 404)
	_, err = s.UpdateBullets(ctx, id, session, []string{" "})
	isAPIErrorWithCode(t, err, 400)
	if err := s.DeleteSession(ctx, id, session); err != nil {
		t.Fatal(err)
	}
}

func TestNotesExpansionQueueAndStorage(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "queued", true: "queue-failure-rollback"}[fail], func(t *testing.T) {
			s, r, storage, q, id, client, _ := newNotesExpansionService(t)
			q.fail = fail
			ctx := context.Background()
			url, key, err := s.PresignNoteUpload(ctx, id, "../../other/private.jpg", "image/webp")
			if err != nil || url == "" || key != storage.put || !strings.HasPrefix(key, "notes/"+id.String()+"/") || !strings.HasSuffix(key, ".webp") || strings.Contains(key, "private") {
				t.Fatalf("presign: %q %q %v", url, key, err)
			}
			got, err := s.CreateSession(ctx, id, CreateSessionInput{ExternalClientID: &client, ImageKey: key})
			if len(q.jobs) != 1 || q.jobs[0].TherapistID != id || q.jobs[0].ImageKey != key {
				t.Fatalf("job: %+v", q.jobs)
			}
			if fail {
				if !errors.Is(err, notesExpansionFailure) || got != nil || r.deletes != 1 || len(r.sessions) != 1 {
					t.Fatalf("rollback: %+v %v deletes=%d rows=%d", got, err, r.deletes, len(r.sessions))
				}
			} else {
				if err != nil || got.Status != models.ClientSessionPending || got.ID != q.jobs[0].SessionID {
					t.Fatalf("created: %+v %v", got, err)
				}
				storage.fail = true
				if err := s.DeleteSession(ctx, id, got.ID); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(storage.deleted, []string{key}) {
					t.Fatalf("cleanup: %v", storage.deleted)
				}
			}
			storage.fail = true
			url, key, err = s.PresignNoteUpload(ctx, id, "x", "image/png")
			if !errors.Is(err, notesExpansionFailure) || url != "" || key != "" {
				t.Fatalf("failed presign leaked values: %q %q %v", url, key, err)
			}
		})
	}
}

func TestNotesExpansionConsentAndValidation(t *testing.T) {
	s, r, _, q, id, client, _ := newNotesExpansionService(t)
	linked := uuid.New()
	ctx := context.Background()
	for _, input := range []CreateSessionInput{
		{ExternalClientID: &client, LinkedClientID: &linked, Bullets: []string{"a"}},
		{ExternalClientID: &client},
		{ExternalClientID: &client, ImageKey: "photo", Bullets: []string{"a"}},
		{ExternalClientID: &client, SessionDate: "2026-99-99", Bullets: []string{"a"}},
		{ExternalClientID: &client, ImageKey: "notes/" + uuid.NewString() + "/photo.jpg"},
		{ExternalClientID: &client, Bullets: []string{"  "}},
	} {
		_, err := s.CreateSession(ctx, id, input)
		isAPIErrorWithCode(t, err, 400)
	}
	_, err := s.CreateSession(ctx, uuid.New(), CreateSessionInput{ExternalClientID: &client, Bullets: []string{"a"}})
	isAPIErrorWithCode(t, err, 404)
	_, err = s.CreateSession(ctx, id, CreateSessionInput{LinkedClientID: &linked, Bullets: []string{"a"}})
	isAPIErrorWithCode(t, err, 404)
	s.links = notesExpansionLinks{fail: true}
	_, err = s.CreateSession(ctx, id, CreateSessionInput{LinkedClientID: &linked, Bullets: []string{"a"}})
	if !errors.Is(err, notesExpansionFailure) || len(r.sessions) != 1 || len(q.jobs) != 0 {
		t.Fatalf("consent failure side effects: %v", err)
	}
	bullets := append([]string{" ", strings.Repeat("x", 2001)}, make([]string, 110)...)
	for i := 2; i < len(bullets); i++ {
		bullets[i] = "entry"
	}
	clean := cleanBullets(bullets)
	if len(clean) != 100 || len(clean[0]) != 2000 {
		t.Fatalf("bounds: %d %d", len(clean), len(clean[0]))
	}
}

func TestNotesExpansionCipherFailuresAndEmptyValues(t *testing.T) {
	ctx, id := context.Background(), uuid.New()
	r := &notesExpansionRepo{fakeNotesRepo: newFakeNotesRepo()}
	master, err := pkgcrypto.NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	c := NewNotesCipher(master, r)
	r.fail = "key"
	if got, err := c.EncryptField(ctx, id, "b", ""); err != nil || got != nil {
		t.Fatalf("empty field: %v %v", got, err)
	}
	if got, err := c.DecryptField(ctx, id, "b", nil); err != nil || got != "" {
		t.Fatalf("empty cipher: %q %v", got, err)
	}
	if got, err := c.EncryptBullets(ctx, id, "b", nil); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if got, err := c.DecryptBullets(ctx, id, "b", nil); err != nil || got == nil || len(got) != 0 {
		t.Fatal(got, err)
	}
	for _, op := range []string{"key", "insert-key"} {
		r.fail = op
		if _, err := c.EncryptField(ctx, id, "b", "secret"); !errors.Is(err, notesExpansionFailure) {
			t.Fatal(err)
		}
		if _, err := c.EncryptBullets(ctx, id, "b", []string{"secret"}); !errors.Is(err, notesExpansionFailure) {
			t.Fatal(err)
		}
		if _, err := c.DecryptField(ctx, id, "b", []byte("broken")); !errors.Is(err, notesExpansionFailure) {
			t.Fatal(err)
		}
	}
	r.fail = ""
	if _, err := NewNotesCipher([]byte("invalid"), r).EncryptField(ctx, uuid.New(), "b", "secret"); err == nil {
		t.Fatal("invalid master key accepted")
	}
	r.keys[id] = []byte("bad wrapped key")
	if _, err := c.EncryptField(ctx, id, "b", "secret"); err == nil {
		t.Fatal("invalid wrapped key accepted")
	}
	delete(r.keys, id)
	ct, err := c.EncryptField(ctx, id, "b", "not JSON")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DecryptBullets(ctx, id, "b", ct); err == nil || !strings.Contains(err.Error(), "unmarshal bullets") {
		t.Fatal(err)
	}
	ct[len(ct)-1] ^= 1
	if _, err := c.DecryptBullets(ctx, id, "b", ct); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}
}

func TestNotesExpansionEncryptionAndProjectionFailures(t *testing.T) {
	for _, op := range []string{"create-client", "update-client", "create-session", "update-bullets"} {
		t.Run(op, func(t *testing.T) {
			s, r, _, _, id, client, session := newNotesExpansionService(t)
			r.fail = "key"
			ctx := context.Background()
			var err error
			switch op {
			case "create-client":
				_, err = s.CreateExternalClient(ctx, id, "Name", "")
			case "update-client":
				name := "Name"
				_, err = s.UpdateExternalClient(ctx, id, client, UpdateExternalClientInput{Name: &name})
			case "create-session":
				_, err = s.CreateSession(ctx, id, CreateSessionInput{ExternalClientID: &client, Bullets: []string{"note"}})
			case "update-bullets":
				_, err = s.UpdateBullets(ctx, id, session, []string{"note"})
			}
			if !errors.Is(err, notesExpansionFailure) {
				t.Fatal(err)
			}
		})
	}
	for _, field := range []string{"name", "raw", "bullets", "summary"} {
		t.Run(field, func(t *testing.T) {
			s, r, _, _, id, client, session := newNotesExpansionService(t)
			ctx := context.Background()
			if field == "name" {
				r.clients[client].NameEnc = []byte("corrupt")
				if _, err := s.GetExternalClient(ctx, id, client); err == nil {
					t.Fatal("corrupt name returned")
				}
				if _, err := s.ListExternalClients(ctx, id, false); err == nil {
					t.Fatal("corrupt name listed")
				}
				return
			}
			row := r.sessions[session]
			switch field {
			case "raw":
				row.RawTextEnc = []byte("corrupt")
			case "bullets":
				row.BulletsEnc = []byte("corrupt")
			case "summary":
				row.SummaryEnc = []byte("corrupt")
			}
			if _, err := s.GetSession(ctx, id, session); err == nil || !strings.Contains(err.Error(), "decrypt "+map[string]string{"raw": "raw text", "bullets": "bullets", "summary": "summary"}[field]) {
				t.Fatal(err)
			}
			if _, err := s.ListSessions(ctx, id, nil, nil, 50); err == nil {
				t.Fatal("corrupt session listed")
			}
		})
	}
	s, r, _, _, id, _, session := newNotesExpansionService(t)
	ctx := context.Background()
	row := r.sessions[session]
	row.RawTextEnc, _ = s.cipher.EncryptField(ctx, id, BindSessionRawText(session), "raw clinical text")
	row.SummaryEnc, _ = s.cipher.EncryptField(ctx, id, BindSessionSummary(session), "clinical summary")
	msg := "OCR warning"
	row.ErrorMsg = &msg
	got, err := s.GetSession(ctx, id, session)
	if err != nil || got.RawText != "raw clinical text" || got.Summary != "clinical summary" || got.ErrorMsg != msg || got.SessionDate != "2026-10-14" || s.Cipher() != s.cipher {
		t.Fatalf("projection: %+v %v", got, err)
	}
}

func TestNotesExpansionSummaryFailures(t *testing.T) {
	for _, stage := range []string{"empty", "decrypt", "ai", "encrypt"} {
		t.Run(stage, func(t *testing.T) {
			s, r, _, _, id, _, session := newNotesExpansionService(t)
			row := r.sessions[session]
			switch stage {
			case "empty":
				row.BulletsEnc = nil
			case "decrypt":
				row.BulletsEnc = []byte("invalid")
			case "ai":
				s.claude = notesExpansionSummary{}
			case "encrypt":
				s.claude = notesExpansionSummary{run: func() { r.fail = "key" }}
			}
			_, err := s.Summarize(context.Background(), id, session)
			if stage == "empty" {
				isAPIErrorWithCode(t, err, 409)
			} else if err == nil {
				t.Fatal("expected summary failure")
			}
			if stage == "ai" || stage == "encrypt" {
				if !errors.Is(err, notesExpansionFailure) {
					t.Fatal(err)
				}
			}
			if row.SummaryEnc != nil {
				t.Fatal("failed summary was persisted")
			}
		})
	}
}

func TestNotesExpansionDefaultSessionDate(t *testing.T) {
	s, _, _, _, id, client, _ := newNotesExpansionService(t)
	before := time.Now().UTC().Format("2006-01-02")
	got, err := s.CreateSession(context.Background(), id, CreateSessionInput{ExternalClientID: &client, Bullets: []string{"note"}})
	after := time.Now().UTC().Format("2006-01-02")
	if err != nil || (got.SessionDate != before && got.SessionDate != after) {
		t.Fatalf("default date: %+v %v", got, err)
	}
}
