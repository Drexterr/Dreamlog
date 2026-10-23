package services

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

type fakeAccountStore struct {
	prefixes  []string
	prefixErr error
	deleteErr error
	calls     []string
}

func (f *fakeAccountStore) StoragePrefixes(_ context.Context, _ uuid.UUID) ([]string, error) {
	f.calls = append(f.calls, "prefixes")
	return f.prefixes, f.prefixErr
}

func (f *fakeAccountStore) Delete(_ context.Context, _ uuid.UUID) error {
	f.calls = append(f.calls, "delete")
	return f.deleteErr
}

type fakePrefixDeleter struct {
	deleted []string
	err     error
	order   *[]string
}

func (f *fakePrefixDeleter) DeletePrefix(_ context.Context, p string) (int, error) {
	f.deleted = append(f.deleted, p)
	if f.order != nil {
		*f.order = append(*f.order, "storage")
	}
	return 1, f.err
}

// N7: account deletion erases the user's storage prefixes after the rows.
func TestDeleteAccount_RemovesRowsThenStoragePrefixes(t *testing.T) {
	store := &fakeAccountStore{prefixes: []string{"audio/u/", "therapy/u/", "tts/s1/", "notes/t1/"}}
	objects := &fakePrefixDeleter{order: &store.calls}
	if err := deleteAccount(context.Background(), store, objects, nil, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(objects.deleted, store.prefixes) {
		t.Errorf("every prefix must be erased: %v", objects.deleted)
	}
	if store.calls[0] != "prefixes" || store.calls[1] != "delete" || store.calls[2] != "storage" {
		t.Errorf("order must be prefixes → row delete → storage, got %v", store.calls)
	}
}

// A failed row delete leaves storage alone and reports the error.
func TestDeleteAccount_RowDeleteFails_StorageUntouched(t *testing.T) {
	store := &fakeAccountStore{prefixes: []string{"audio/u/"}, deleteErr: errors.New("db down")}
	objects := &fakePrefixDeleter{}
	if err := deleteAccount(context.Background(), store, objects, nil, uuid.New()); err == nil {
		t.Fatal("row delete failure must be returned")
	}
	if len(objects.deleted) != 0 {
		t.Error("storage must not be erased when the account was not deleted")
	}
}

// Storage trouble never fails a deletion the user asked for.
func TestDeleteAccount_StorageFailure_StillSucceeds(t *testing.T) {
	store := &fakeAccountStore{prefixes: []string{"audio/u/", "therapy/u/"}}
	objects := &fakePrefixDeleter{err: errors.New("r2 unavailable")}
	if err := deleteAccount(context.Background(), store, objects, nil, uuid.New()); err != nil {
		t.Fatalf("storage failure must not fail account deletion: %v", err)
	}
	if len(objects.deleted) != 2 {
		t.Errorf("every prefix must still be attempted, got %v", objects.deleted)
	}
}

// Without a storage client (tests, dev) deletion still works.
func TestDeleteAccount_NoStorage(t *testing.T) {
	store := &fakeAccountStore{}
	if err := deleteAccount(context.Background(), store, nil, nil, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.calls, []string{"delete"}) {
		t.Errorf("only the row delete runs without storage: %v", store.calls)
	}
}
