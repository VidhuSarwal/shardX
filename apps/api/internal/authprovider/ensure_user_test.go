package authprovider

import (
	"context"
	"testing"

	"SE/internal/metadatastore"
	"SE/internal/models"
)

type fakeUserStore struct {
	metadatastore.MetadataStore // unimplemented methods panic if reached
	users                       map[string]*models.User
}

func (f *fakeUserStore) FindUserByEmail(_ context.Context, email string) (*models.User, error) {
	return f.users[email], nil
}
func (f *fakeUserStore) CreateUser(_ context.Context, u *models.User) error {
	f.users[u.Email] = u
	return nil
}

func TestEnsureUser_CreatesOnceWithSubDerivedID(t *testing.T) {
	orig := metadatastore.Active
	defer func() { metadatastore.Active = orig }()
	fs := &fakeUserStore{users: map[string]*models.User{}}
	metadatastore.Active = fs

	id := subToObjectID("11111111-2222-3333-4444-555555555555")
	ensureUser(context.Background(), id, "a@b.c")
	ensureUser(context.Background(), id, "a@b.c")

	if len(fs.users) != 1 || fs.users["a@b.c"].ID != id {
		t.Fatalf("users = %+v", fs.users)
	}

	metadatastore.Active = nil
	ensureUser(context.Background(), id, "x@y.z") // must not panic
}
