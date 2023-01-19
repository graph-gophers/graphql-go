// package user contains a naive implementation of an user with roles.
// Each user can be assigned roles and added to/retrieved from context.
package user

import (
	"context"
)

type userKey string

const contextKey userKey = "user"

type User struct {
	ID    string
	Roles map[string]struct{}
}

func (u *User) AddRole(r string) {
	if u.Roles == nil {
		u.Roles = map[string]struct{}{}
	}
	u.Roles[r] = struct{}{}
}

func (u *User) HasRole(r string) bool {
	_, ok := u.Roles[r]
	return ok
}

func AddToContext(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, contextKey, u)
}

func FromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(contextKey).(*User)
	return u, ok
}
