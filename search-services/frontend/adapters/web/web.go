package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"search-service/frontend/core"
	"time"
)

const (
	paramPhrase = "phrase"

	cookieName = "jwt_token"
)

func encodeReply(w io.Writer, reply any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(reply); err != nil {
		return fmt.Errorf("could not encode reply: %v", err)
	}
	return nil
}

func NewPingHandler(log *slog.Logger, pinger core.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := pinger.Ping(r.Context())
		if err != nil {
			if errors.Is(err, core.ErrServiceUnavailable) {
				log.Debug("ping endpoint unavailable")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			} else {
				log.Warn("ping endpoint failed", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

func NewLoginHandler(log *slog.Logger, auth core.Authenticator, tokenTTL time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var login struct {
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		tokenString, err := auth.CreateToken(login.Name, login.Password)
		if err != nil {
			if errors.Is(err, core.ErrInvalidCredentials) {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			} else {
				log.Error("failed to create token", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		cookie := &http.Cookie{
			Name:     cookieName,
			Value:    tokenString,
			Path:     "/",
			MaxAge:   int(tokenTTL.Seconds()),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)
	}
}

func NewSearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get(paramPhrase)
		if phrase == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		reply, err := searcher.Search(r.Context(), phrase)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrBadArguments):
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			case errors.Is(err, core.ErrServiceUnavailable):
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			default:
				log.Warn("service search failed", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

type statistics struct {
	Stats  core.UpdateStats  `json:"stats"`
	Status core.UpdateStatus `json:"status"`
}

func NewStatisticsHandler(log *slog.Logger, statsProvider core.UpdateStatsProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := statsProvider.GetUpdateStats(r.Context())
		if err != nil {
			if errors.Is(err, core.ErrServiceUnavailable) {
				log.Debug("stats endpoint unavailable")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			} else {
				log.Warn("stats endpoint failed", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		status, err := statsProvider.GetUpdateStatus(r.Context())
		if err != nil {
			if errors.Is(err, core.ErrServiceUnavailable) {
				log.Debug("status endpoint unavailable")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			} else {
				log.Warn("status endpoint failed", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		reply := statistics{
			Stats:  stats,
			Status: status,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := updater.Update(r.Context()); err != nil {
			switch {
			case errors.Is(err, core.ErrServiceUnavailable):
				log.Debug("service update unavailable")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			case errors.Is(err, core.ErrAlreadyExists):
				log.Debug("service update already running")
				http.Error(w, http.StatusText(http.StatusAccepted), http.StatusAccepted)
			default:
				log.Warn("service update failed", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := updater.Drop(r.Context()); err != nil {
			if errors.Is(err, core.ErrServiceUnavailable) {
				log.Debug("service update unavailable")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			} else {
				log.Warn("service update failed", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}
	}
}

func NewPageHandler(fs fs.FS, filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, fs, filename)
	}
}
