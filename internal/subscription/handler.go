package subscription

import (
	"encoding/json"
	"errors"
	"net/http"
	"relay-hook/internal/auth"
	"relay-hook/internal/event"
)

func GetAllHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := auth.KeyFromContext(r.Context())
		if !ok {
			http.Error(w, ErrUnauthorized.Error(), http.StatusInternalServerError)
			return
		}

		subs, err := svc.GetAll(r.Context(), key.UserID)
		if err != nil {
			http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subs)
	}
}

func RegisterHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := auth.KeyFromContext(r.Context())
		if !ok {
			http.Error(w, ErrUnauthorized.Error(), http.StatusInternalServerError)
			return
		}
		var body struct {
			EventType   event.Type `json:"event_type"`
			EndpointURL string     `json:"endpoint_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
			return
		}
		permits := key.Permits(string(body.EventType))
		if !permits {
			http.Error(w, ErrNotAllowed.Error(), http.StatusForbidden)
			return
		}

		sub, err := svc.Register(r.Context(), key.UserID, body.EventType, body.EndpointURL)
		if err != nil {
			http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sub)
	}
}

func UpdateSubscriberHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := auth.KeyFromContext(r.Context())
		if !ok {
			http.Error(w, ErrUnauthorized.Error(), http.StatusInternalServerError)
			return
		}

		id := r.PathValue("id")
		if id == "" {
			http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
			return
		}
		var body struct {
			EndpointURL string `json:"endpoint_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, ErrInvalidPayload.Error(), http.StatusBadRequest)
			return
		}

		err := svc.repo.UpdateEndpoint(r.Context(), id, key.UserID, body.EndpointURL)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
			} else {
				http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func DeleteHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := auth.KeyFromContext(r.Context())
		if !ok {
			http.Error(w, ErrUnauthorized.Error(), http.StatusInternalServerError)
			return
		}

		id := r.PathValue("id")
		if id == "" {
			http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
			return
		}
		if err := svc.Delete(r.Context(), id, key.UserID); err != nil {
			if errors.Is(err, ErrNotFound) {
				http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
			} else {
				http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func DeleteSubscriberHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := auth.KeyFromContext(r.Context())
		if !ok {
			http.Error(w, ErrUnauthorized.Error(), http.StatusInternalServerError)
			return
		}

		id := r.PathValue("id")
		if id == "" {
			http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
			return
		}
		if err := svc.DeleteSubscriber(r.Context(), id, key.UserID); err != nil {
			if errors.Is(err, ErrNotFound) {
				http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
			} else {
				http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
