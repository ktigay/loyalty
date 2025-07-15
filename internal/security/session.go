package security

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/ktigay/loyalty/internal/entity"
)

type ctxKey int

const (
	userIdentityCtxKey ctxKey = iota

	userIdentityKey = "userIdentityKey"
	sessionTokenKey = "sessionToken"
)

// sessionStore Стор сессии пользователя.
type sessionStore interface {
	Get(r *http.Request, name string) (*sessions.Session, error)
	Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error
	New(r *http.Request, name string) (*sessions.Session, error)
}

// SessionWrapper Врапер сессии.
type SessionWrapper struct {
	store sessionStore
}

// GetIdentity Сессия из реквеста.
func (s *SessionWrapper) GetIdentity(r *http.Request) (*entity.Identity, error) {
	var (
		sess     *sessions.Session
		identity *entity.Identity
		err      error
		ok       bool
		val      interface{}
	)
	if sess, err = s.store.Get(r, sessionTokenKey); err != nil {
		return nil, err
	}

	val, ok = sess.Values[userIdentityKey]
	if !ok {
		return nil, ErrMissingIdentity
	}
	if identity, ok = val.(*entity.Identity); !ok {
		return nil, ErrMissingIdentity
	}

	return identity, nil
}

// SetIdentity Сохранить сессию.
func (s *SessionWrapper) SetIdentity(r *http.Request, w http.ResponseWriter, identity *entity.Identity) error {
	sess, err := s.store.Get(r, sessionTokenKey)
	if err != nil {
		return err
	}
	sess.Values[userIdentityKey] = identity
	return s.store.Save(r, w, sess)
}

// NewSessionWrapper Конструктор.
func NewSessionWrapper(secret string) *SessionWrapper {
	return &SessionWrapper{
		store: sessions.NewCookieStore([]byte(secret)),
	}
}
