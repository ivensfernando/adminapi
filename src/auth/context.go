package auth

import (
	"context"
	"vsC1Y2025V01/src/model"
)

type contextKey string

const UserKey contextKey = "user"

func GetUserFromContext(ctx context.Context) (*model.User, bool) {
	user, ok := ctx.Value(UserKey).(*model.User)
	return user, ok
}
