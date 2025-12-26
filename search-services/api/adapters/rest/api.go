package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"search-service/api/core"
	"strconv"
)

const (
	paramPhrase = "phrase"
	paramLimit  = "limit"
	searchLimit = 10
)

func encodeReply(w io.Writer, reply any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(reply); err != nil {
		return fmt.Errorf("could not encode reply: %v", err)
	}
	return nil
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply := core.PingResponse{
			Replies: make(map[string]core.PingStatus, len(pingers)),
		}
		for name, pinger := range pingers {
			err := pinger.Ping(r.Context())
			if err == nil {
				reply.Replies[name] = core.StatusPingOK
				continue
			}
			if errors.Is(err, core.ErrServiceUnavailable) {
				log.Debug("service unavailable", "service", name)
			} else {
				log.Warn("service ping failed", "service", name, "error", err)
			}
			reply.Replies[name] = core.StatusPingUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

func NewLoginHandler(log *slog.Logger, auth core.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var login core.LoginRequest
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
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(tokenString))
	}
}

func NewSearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get(paramPhrase)
		if phrase == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		limitStr := r.URL.Query().Get(paramLimit)
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			if limitStr != "" {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			limit = searchLimit
		}
		if limit <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		comics, err := searcher.Search(r.Context(), phrase, int64(limit))
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
		if err := encodeReply(w, core.SearchResult{Comics: comics, Total: int64(len(comics))}); err != nil {
			log.Error("failed to encode", "error", err)
		}
	}
}

func NewISearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get(paramPhrase)
		if phrase == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		limitStr := r.URL.Query().Get(paramLimit)
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			if limitStr != "" {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			limit = searchLimit
		}
		if limit <= 0 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		comics, err := searcher.ISearch(r.Context(), phrase, int64(limit))
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
		if err := encodeReply(w, core.SearchResult{Comics: comics, Total: int64(len(comics))}); err != nil {
			log.Error("failed to encode", "error", err)
		}
	}
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := updater.Stats(r.Context())
		if err != nil {
			if errors.Is(err, core.ErrServiceUnavailable) {
				log.Debug("service update unavailable")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			} else {
				log.Warn("service update failed", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := encodeReply(w, stats); err != nil {
			log.Error("failed to encode", "error", err)
		}
	}
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := updater.Status(r.Context())
		if err != nil {
			if errors.Is(err, core.ErrServiceUnavailable) {
				log.Debug("service update unavailable")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			} else {
				log.Warn("service update failed", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := encodeReply(w, core.UpdateStatusResponse{Status: status}); err != nil {
			log.Error("failed to encode", "error", err)
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
			switch {
			case errors.Is(err, core.ErrServiceUnavailable):
				log.Debug("service drop unavailable")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			case errors.Is(err, core.ErrAlreadyExists):
				log.Debug("service drop already running")
				http.Error(w, http.StatusText(http.StatusAccepted), http.StatusAccepted)
			default:
				log.Warn("service drop failed", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}
	}
}
