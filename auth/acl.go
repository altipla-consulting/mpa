package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	pbsecurity "github.com/lavozdealmeria/glider/protos/security"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"libs.altipla.consulting/collections"
	"libs.altipla.consulting/connect"
	"libs.altipla.consulting/env"
	"libs.altipla.consulting/errors"
	"libs.altipla.consulting/routing"
)

var client pbsecurity.SecurityServiceClient

func init() {
	conn, err := connect.Remote(connect.Endpoint{Remote: "glider-ubirgsu2wa-ew.a.run.app"})
	if err != nil {
		log.Fatal(err)
	}
	client = pbsecurity.NewSecurityServiceClient(conn)
}

type ACLRestriction func(r *aclControl)

func Perm(perm string) ACLRestriction {
	return func(r *aclControl) {
		r.perms = append(r.perms, perm)
	}
}

type aclControl struct {
	checked      bool
	perms        []string
	enabledPerms []string
	token        string
}

type contextKey int

const (
	aclControlKey = contextKey(1)
)

var (
	errRequiresLogin = errors.New("acl: requires login")
	errPermRequired  = errors.New("acl: permission required")
)

func ACL(r *http.Request, restrictions ...ACLRestriction) (*http.Request, error) {
	control, ok := r.Context().Value(aclControlKey).(*aclControl)
	if !ok {
		return nil, routing.Unauthorized("acl is not configured in this handler")
	}

	if control.checked {
		return nil, errors.Errorf("cannot check acl twice")
	}

	for _, restriction := range restrictions {
		restriction(control)
	}

	if env.IsLocal() {
		deny := strings.Split(r.URL.Query().Get("$$deny"), ",")
		log.WithFields(log.Fields{
			"permissions":    control.perms,
			"token":          control.token,
			"simulated-deny": deny,
		}).Debug("Simulated check against Glider permissions")
		for _, perm := range control.perms {
			if collections.HasString(deny, perm) {
				return nil, routing.Unauthorizedf("permissions required: %s", control.perms)
			}
		}
	} else {
		checkReq := &pbsecurity.CheckRequest{
			BearerToken: control.token,
			Permissions: control.perms,
		}
		reply, err := client.Check(r.Context(), checkReq)
		if err != nil {
			if status.Code(err) == codes.InvalidArgument {
				return nil, errors.Trace(errRequiresLogin)
			}
			return nil, errors.Trace(err)
		}
		if !reply.CanAccess {
			return nil, routing.Unauthorizedf("permissions required: %s", control.perms)
		}
		control.enabledPerms = reply.Permissions
	}

	control.checked = true

	return r, nil
}

func RequireACL(handler routing.Handler) routing.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		control, ok := r.Context().Value(aclControlKey).(*aclControl)
		if ok {
			return errors.Errorf("cannot check acl twice")
		}

		control = new(aclControl)
		r = r.WithContext(context.WithValue(r.Context(), aclControlKey, control))

		cookie, err := r.Cookie("token")
		if err != nil && !errors.Is(err, http.ErrNoCookie) {
			return errors.Trace(err)
		} else if err != nil {
			return errors.Trace(redirectToLogin(w, r))
		}
		control.token = cookie.Value

		if err := handler(w, r); err != nil {
			if errors.Is(err, errRequiresLogin) {
				return errors.Trace(redirectToLogin(w, r))
			}
			return errors.Trace(err)
		}

		if !control.checked {
			return routing.Unauthorized("acl is not configured in this handler")
		}

		return nil
	}
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) error {
	next := r.URL.Path
	if r.URL.RawQuery != "" {
		next += "?" + r.URL.RawQuery
	}
	q := make(url.Values)
	q.Set("next", next)
	u := &url.URL{
		Path:     "/accounts/login",
		RawQuery: q.Encode(),
	}
	http.Redirect(w, r, u.String(), http.StatusFound)
	return nil
}

func HasPerm(r *http.Request, perm string) bool {
	control, ok := r.Context().Value(aclControlKey).(*aclControl)
	if !ok {
		panic("acl is not configured in this handler")
	}
	if !control.checked {
		panic("check permissions with ACL() before calling HasPerm()")
	}

	if env.IsLocal() {
		deny := strings.Split(r.URL.Query().Get("$$deny"), ",")
		return !collections.HasString(deny, perm)
	}

	return collections.HasString(control.enabledPerms, perm)
}
