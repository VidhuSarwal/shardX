package handlers

import (
	"SE/internal/metadatastore"
	"encoding/json"
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DriveAccountsHandler holds dependencies for drive-account related
// handlers. It accepts the metadatastore.MetadataStore interface rather
// than calling internal/store directly, so it can be backed by any
// MetadataStore implementation (currently only MongoStore).
type DriveAccountsHandler struct {
	Store metadatastore.MetadataStore
}

// NewDriveAccountsHandler constructs a DriveAccountsHandler.
func NewDriveAccountsHandler(s metadatastore.MetadataStore) *DriveAccountsHandler {
	return &DriveAccountsHandler{Store: s}
}

// ListDriveAccounts - GET /api/drive/accounts
func (h *DriveAccountsHandler) ListDriveAccounts(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)

	accts, err := h.Store.ListUserDriveAccounts(r.Context(), userID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// do not return encrypted token in response
	type DriveAccountOut struct {
		ID          primitive.ObjectID `json:"id"`
		Provider    string             `json:"provider"`
		DisplayName string             `json:"display_name"`
		CreatedAt   interface{}        `json:"created_at"`
	}

	out := make([]DriveAccountOut, 0, len(accts))
	for _, a := range accts {
		out = append(out, DriveAccountOut{
			ID:          a.ID,
			Provider:    a.Provider,
			DisplayName: a.DisplayName,
			CreatedAt:   a.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
