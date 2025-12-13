package auth

import (
	"adminapi/src/model"
	"context"
)

type contextKey string

const UserKey contextKey = "user"

func GetUserFromContext(ctx context.Context) (*model.User, bool) {
	user, ok := ctx.Value(UserKey).(*model.User)
	return user, ok
}
