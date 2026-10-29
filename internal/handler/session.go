package handler

import (
	"net/http"

	"github.com/gorilla/sessions"
)

var sessionStore = sessions.NewCookieStore(
	[]byte("change-this-session-secret"),
)

const sessionName = "leadflow_session"

func setSession(
	w http.ResponseWriter,
	r *http.Request,
	userID int64,
) error {
	session, err := sessionStore.Get(r, sessionName)
	if err != nil {
		return err
	}

	session.Values["user_id"] = userID
	session.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 7,
	}

	return session.Save(r, w)
}

func getUserIDFromSession(
	r *http.Request,
) (int64, bool) {
	session, err := sessionStore.Get(r, sessionName)
	if err != nil {
		return 0, false
	}

	value, ok := session.Values["user_id"].(int64)

	return value, ok
}

func clearSession(
	w http.ResponseWriter,
	r *http.Request,

) error {
	session, err := sessionStore.Get(r, sessionName)
	if err != nil {
		return err
	}

	session.Options.MaxAge = -1

	return session.Save(r, w)
}
