package security

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ktigay/loyalty/internal/entity"
)

const (
	subjectKey          = "sub"
	authorizationHeader = "Authorization"
)

// JWTWrapper Обертка для JWT.
type JWTWrapper struct {
	secret []byte
}

// GetIdentity Возвращает идентификатор пользователя из реквеста.
func (j *JWTWrapper) GetIdentity(r *http.Request) (*entity.Identity, error) {
	var (
		identity *entity.Identity
		err      error
	)
	identity, err = parseToken[entity.Identity](
		j.secret,
		r.Header.Get(authorizationHeader),
	)
	if err != nil {
		return nil, err
	}

	return identity, nil
}

// SetIdentity Устанавливает идентификатор пользователя в респонс.
func (j *JWTWrapper) SetIdentity(_ *http.Request, w http.ResponseWriter, identity *entity.Identity) error {
	token, err := generateToken(j.secret, *identity)
	if err != nil {
		return err
	}

	w.Header().Set(authorizationHeader, token)
	return nil
}

// NewJWTWrapper Конструктор.
func NewJWTWrapper(secret string) *JWTWrapper {
	return &JWTWrapper{
		secret: []byte(secret),
	}
}

func generateToken(secret []byte, payload any) (string, error) {
	v, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		subjectKey: string(v),
	})
	return token.SignedString(secret)
}

func parseToken[P any](secret []byte, tokenString string) (*P, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if len(tokenString) == 0 {
		return nil, ErrMissingIdentity
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		var subj string

		subj, err = claims.GetSubject()
		if err != nil {
			return nil, ErrMissingIdentity
		}

		var resp P
		if err = json.Unmarshal([]byte(subj), &resp); err != nil {
			return nil, err
		}

		return &resp, nil
	}

	return nil, ErrMissingIdentity
}
