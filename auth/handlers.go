package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/segmentio/ksuid"
	"golang.org/x/oauth2"
	"libs.altipla.consulting/cloudrun"
	"libs.altipla.consulting/env"
	"libs.altipla.consulting/errors"
	"libs.altipla.consulting/routing"
)

type authConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func Configure(r *cloudrun.WebServer) {
	acl := r.PathPrefix("/accounts")
	acl.Get("/login", loginHandler)
	acl.Get("/logout", logoutHandler)
}

func loginHandler(w http.ResponseWriter, r *http.Request) error {
	q := make(url.Values)
	q.Set("next", r.FormValue("next"))
	u := &url.URL{
		Scheme:   "https",
		Host:     r.Host,
		Path:     "/accounts/login",
		RawQuery: q.Encode(),
	}

	var cnf authConfig
	if err := json.Unmarshal([]byte(os.Getenv("AUTH0_LOGIN_CLIENT")), &cnf); err != nil {
		return errors.Trace(err)
	}
	loginConfig := &oauth2.Config{
		ClientID:     cnf.ClientID,
		ClientSecret: cnf.ClientSecret,
		Scopes:       []string{"openid"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://lavoz.eu.auth0.com/authorize",
			TokenURL: "https://lavoz.eu.auth0.com/oauth/token",
		},
		RedirectURL: u.String(),
	}

	code := r.FormValue("code")
	if code != "" {
		token, err := loginConfig.Exchange(r.Context(), code)
		if err != nil {
			return errors.Trace(err)
		}

		state, err := r.Cookie("login-state")
		if err != nil {
			if err == http.ErrNoCookie {
				goToNext(w, r)
				return nil
			}
			return errors.Trace(err)
		}
		if state.Value != r.FormValue("state") {
			return routing.Unauthorized("bad state")
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "login-state",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			Secure:   true,
			HttpOnly: true,
		})

		http.SetCookie(w, &http.Cookie{
			Name:    "token",
			Value:   token.AccessToken,
			Path:    "/",
			Expires: token.Expiry,
			Secure:  true,
		})

		goToNext(w, r)
		return nil
	}

	state := ksuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     "login-state",
		Value:    state,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	})
	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("audience", "https://glider.lavozdealmeria.com/"),
	}
	if r.FormValue("prompt") == "login" {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", "login"))
	}
	http.Redirect(w, r, loginConfig.AuthCodeURL(state, opts...), http.StatusFound)

	return nil
}

func goToNext(w http.ResponseWriter, r *http.Request) {
	next := "/"
	if r.FormValue("next") != "/" {
		u, err := url.Parse(r.FormValue("next"))
		if err == nil {
			next = u.Path
			if u.RawQuery != "" {
				next += "?" + u.RawQuery
			}
		}
	}
	http.Redirect(w, r, next, http.StatusFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
		Secure:  true,
	})

	u, err := url.Parse("https://lavoz.eu.auth0.com/v2/logout")
	if err != nil {
		return errors.Trace(err)
	}
	q := make(url.Values)
	if env.IsLocal() {
		q.Set("returnTo", "https://"+r.Host+"/")
	} else {
		q.Set("returnTo", "https://admin.lavozdealmeria.com/")
	}

	var cnf authConfig
	if err := json.Unmarshal([]byte(os.Getenv("AUTH0_LOGIN_CLIENT")), &cnf); err != nil {
		return errors.Trace(err)
	}
	q.Set("client_id", cnf.ClientID)

	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
	return nil
}
