package bootstrap

import (
	"net/http"
	"time"

	"traffic-monitor/internal/config"

	"github.com/gorilla/sessions"
)

func NewSessionStore(cfg config.SessionConfig) *sessions.CookieStore {
	store := sessions.NewCookieStore([]byte(cfg.Secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(cfg.MaxAge / time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Secure,
	}

	return store
}
