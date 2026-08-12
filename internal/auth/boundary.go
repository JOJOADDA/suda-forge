package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type Principal struct {
	Subject string
	Roles   []string
}
type Validator interface {
	Validate(context.Context, string) (Principal, error)
}

var ErrUnauthorized = errors.New("unauthorized")

type Boundary struct {
	Validator Validator
	Enabled   bool
}

func (b Boundary) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !b.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") || b.Validator == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		principal, err := b.Validator.Validate(r.Context(), strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, principal)))
	})
}
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	value, ok := ctx.Value(principalKey{}).(Principal)
	return value, ok
}

type principalKey struct{}
