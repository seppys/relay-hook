package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r *authRequest) validate() error {
	if len(r.Username) == 0 {
		return errors.New("missing username")
	}
	if len(r.Password) == 0 {
		return errors.New("missing password")
	}
	return nil
}

func RegisterHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, ErrInvalidPayload.Error(), http.StatusBadRequest)
		}
		err = req.validate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		user, err := svc.Register(r.Context(), req.Username, req.Password)
		if err != nil {
			if errors.Is(err, ErrExistingUsername) {
				http.Error(w, ErrExistingUsername.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func LoginHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, ErrInvalidPayload.Error(), http.StatusBadRequest)
			return
		}

		token, err := svc.Login(r.Context(), body.Username, body.Password)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				http.Error(w, ErrInvalidCredentials.Error(), http.StatusBadRequest)
				return
			} else if errors.Is(err, ErrInvalidCredentials) {
				http.Error(w, ErrInvalidCredentials.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			return
		}

		var response struct {
			Token string `json:"token"`
		}
		response.Token = token

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func GetKeysHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		keys, err := svc.GetKeys(r.Context(), userID)

		if err != nil {
			if errors.Is(err, ErrKeyNotFound) {
				http.Error(w, ErrKeyNotFound.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keys)
	}
}

type generateKeyRequest struct {
	Role       KeyRole  `json:"role"`
	EventTypes []string `json:"event_types"`
}

func (r *generateKeyRequest) validate() error {
	if len(r.Role) == 0 {
		return errors.New("missing role")
	}
	if !r.Role.Validate() {
		return errors.New("invalid role")
	}
	return nil
}

func GenerateKeyHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		var request generateKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, ErrInvalidPayload.Error(), http.StatusBadRequest)
			return
		}

		if err := request.validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		newKey, err := svc.GenerateKey(r.Context(), userID, request.Role, request.EventTypes)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				http.Error(w, ErrUserNotFound.Error(), http.StatusBadRequest)
			} else if errors.Is(err, ErrRoleNotFound) {
				http.Error(w, ErrRoleNotFound.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newKey)
	}
}

func RemoveKeyHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		keyID := r.PathValue("id")
		if keyID == "" {
			http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
			return
		}

		err := svc.RemoveKey(r.Context(), userID, keyID)
		if err != nil {
			if errors.Is(err, ErrKeyNotFound) {
				http.Error(w, ErrKeyNotFound.Error(), http.StatusBadRequest)
			}
			http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
