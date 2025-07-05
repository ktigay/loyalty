package security

import (
	"context"
	"encoding/gob"
	"fmt"

	"github.com/ktigay/loyalty/internal/entity"
)

var ErrMissingIdentity = fmt.Errorf("missing identity")

// NewContextWithUserIdentity Новый контекст с идентификатором [Identity] пользователя.
func NewContextWithUserIdentity(ctx context.Context, identity *entity.Identity) context.Context {
	return context.WithValue(ctx, userIdentityCtxKey, identity)
}

// UserIdentityFromCtx Идентификатор [Identity] пользователя из контекста.
func UserIdentityFromCtx(ctx context.Context) *entity.Identity {
	if userSess, ok := ctx.Value(userIdentityCtxKey).(*entity.Identity); ok {
		return userSess
	}
	return nil
}

func init() {
	gob.Register(&entity.Identity{})
}
