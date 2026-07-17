package event

import (
	"encoding/json"
	"net/http"
	"relay-hook/internal/auth"
)

func GetAllHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := auth.KeyFromContext(r.Context())
		if !ok {
			http.Error(w, auth.ErrUnauthorized.Error(), http.StatusUnauthorized)
			return
		}
		e, err := svc.GetAll(r.Context(), key.UserID)
		if err != nil {
			http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(e)
	}
}

func CollectHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := auth.KeyFromContext(r.Context())
		if !ok {
			http.Error(w, auth.ErrUnauthorized.Error(), http.StatusUnauthorized)
			return
		}
		var body struct {
			Type    Type            `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, ErrInvalidPayload.Error(), http.StatusBadRequest)
			return
		}

		if ok := key.Permits(string(body.Type)); !ok {
			http.Error(w, ErrNotAllowed.Error(), http.StatusForbidden)
			return
		}

		e, err := svc.Receive(r.Context(), key.UserID, body.Type, body.Payload)
		if err != nil {
			http.Error(w, ErrInternalServer.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(e)
	}
}
