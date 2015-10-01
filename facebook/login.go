package facebook

import (
	"errors"
	"net/http"

	"github.com/dghubble/ctxh"
	"github.com/dghubble/gologin"
	oauth2Login "github.com/dghubble/gologin/oauth2"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// Facebook login errors
var (
	ErrUnableToGetFacebookUser = errors.New("facebook: unable to get Facebook User")
)

// StateHandler checks for a temporary state cookie. If found, the state value
// is read from it and added to the ctx. Otherwise, a temporary state cookie
// is written and the corresponding state value is added to the ctx.
//
// Implements OAuth 2 RFC 6749 10.12 CSRF Protection.
func StateHandler(success ctxh.ContextHandler) ctxh.ContextHandler {
	return oauth2Login.StateHandler(success)
}

// LoginHandler handles Facebook login requests by reading the state value
// from the ctx and redirecting requests to the AuthURL with that state value.
func LoginHandler(config *oauth2.Config, failure ctxh.ContextHandler) ctxh.ContextHandler {
	return oauth2Login.LoginHandler(config, failure)
}

// CallbackHandler handles Facebook redirection URI requests and adds the
// Facebook access token and User to the ctx. If authentication succeeds,
// handling delegates to the success handler, otherwise to the failure
// handler.
func CallbackHandler(config *oauth2.Config, success, failure ctxh.ContextHandler) ctxh.ContextHandler {
	success = facebookHandler(config, success, failure)
	return oauth2Login.CallbackHandler(config, success, failure)
}

// facebookHandler is a ContextHandler that gets the OAuth2 access token from
// the ctx to get the corresponding Facebook User. If successful, the user is
// added to the ctx and the success handler is called. Otherwise, the failure
// handler is called.
func facebookHandler(config *oauth2.Config, success, failure ctxh.ContextHandler) ctxh.ContextHandler {
	if failure == nil {
		failure = gologin.DefaultFailureHandler
	}
	fn := func(ctx context.Context, w http.ResponseWriter, req *http.Request) {
		accessToken, err := oauth2Login.AccessTokenFromContext(ctx)
		if err != nil {
			ctx = gologin.WithError(ctx, err)
			failure.ServeHTTP(ctx, w, req)
			return
		}
		token := &oauth2.Token{AccessToken: accessToken}
		httpClient := config.Client(ctx, token)
		facebookService := newClient(httpClient)
		user, resp, err := facebookService.Me()
		err = validateResponse(user, resp, err)
		if err != nil {
			ctx = gologin.WithError(ctx, err)
			failure.ServeHTTP(ctx, w, req)
			return
		}
		ctx = WithUser(ctx, user)
		success.ServeHTTP(ctx, w, req)
	}
	return ctxh.ContextHandlerFunc(fn)
}

// validateResponse returns an error if the given Facebook User, raw
// http.Response, or error are unexpected. Returns nil if they are valid.
func validateResponse(user *User, resp *http.Response, err error) error {
	if err != nil || resp.StatusCode != http.StatusOK {
		return ErrUnableToGetFacebookUser
	}
	if user == nil || user.ID == "" {
		return ErrUnableToGetFacebookUser
	}
	return nil
}