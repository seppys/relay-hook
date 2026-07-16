package subscription

import (
	"encoding/json"
	"errors"
	"net/http"
	"relay-hook/internal/event"
)

func GetAllHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subs, err := svc.GetAll(r.Context())
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
		var body struct {
			EventType   event.Type `json:"event_type"`
			EndpointURL string     `json:"endpoint_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
			return
		}

		sub, err := svc.Register(r.Context(), body.EventType, body.EndpointURL)
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

		err := svc.repo.UpdateEndpoint(r.Context(), id, body.EndpointURL)
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
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
			return
		}
		if err := svc.Delete(r.Context(), id); err != nil {
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
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
			return
		}
		if err := svc.DeleteSubscriber(r.Context(), id); err != nil {
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
