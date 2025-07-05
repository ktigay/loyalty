package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/handler/user/mocks"
	"github.com/ktigay/loyalty/internal/log"
	"github.com/ktigay/loyalty/internal/security"
	"github.com/ktigay/loyalty/internal/service/user"
	"github.com/stretchr/testify/assert"
)

type fields struct {
	userSv func(ctrl *gomock.Controller) ServiceInterface
	auth   func(ctrl *gomock.Controller) AuthInterface
}

func TestAuthHandler_LoginHandler(t *testing.T) {
	var (
		UUID     = "uuid-1"
		login    = "login"
		password = "password"
	)
	type args struct {
		r []byte
	}
	type want struct {
		wantErr        bool
		wantStatus     int
		wantAuthHeader bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "Request_With_StatusOK",
			fields: fields{
				userSv: func(ctrl *gomock.Controller) ServiceInterface {
					userSv := mocks.NewMockServiceInterface(ctrl)
					userSv.EXPECT().UserByCredentials(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(&entity.User{}, nil)
					return userSv
				},
				auth: func(ctrl *gomock.Controller) AuthInterface {
					auth := security.NewJWTWrapper("secret")
					return auth
				},
			},
			args: args{
				r: func() []byte {
					r, _ := json.Marshal(api.PostUserLoginJSONRequestBody{
						Login:    login,
						Password: password,
					})
					return r
				}(),
			},
			want: want{
				wantStatus:     http.StatusOK,
				wantAuthHeader: true,
			},
		},
		{
			name: "Request_With_Bad_JSON",
			args: args{
				r: []byte(`{"login":1111,"password"`),
			},
			want: want{
				wantStatus: http.StatusBadRequest,
			},
		},
		{
			name: "Request_With_Error_UserNotFound",
			fields: fields{
				userSv: func(ctrl *gomock.Controller) ServiceInterface {
					userSv := mocks.NewMockServiceInterface(ctrl)
					userSv.EXPECT().UserByCredentials(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(nil, user.ErrUserNotFound)
					return userSv
				},
			},
			args: args{
				r: func() []byte {
					r, _ := json.Marshal(api.PostUserLoginJSONRequestBody{
						Login:    login,
						Password: password,
					})
					return r
				}(),
			},
			want: want{
				wantStatus: http.StatusUnauthorized,
			},
		},
		{
			name: "Request_With_Error_Wrong_Password",
			fields: fields{
				userSv: func(ctrl *gomock.Controller) ServiceInterface {
					userSv := mocks.NewMockServiceInterface(ctrl)
					userSv.EXPECT().UserByCredentials(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(nil, user.ErrWrongPassword)
					return userSv
				},
			},
			args: args{
				r: func() []byte {
					r, _ := json.Marshal(api.PostUserLoginJSONRequestBody{
						Login:    login,
						Password: password,
					})
					return r
				}(),
			},
			want: want{
				wantStatus: http.StatusUnauthorized,
			},
		},
		{
			name: "Request_With_Failed_Get_UserByCredentials",
			fields: fields{
				userSv: func(ctrl *gomock.Controller) ServiceInterface {
					userSv := mocks.NewMockServiceInterface(ctrl)
					userSv.EXPECT().UserByCredentials(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(nil, errors.New("some error"))
					return userSv
				},
			},
			args: args{
				r: func() []byte {
					r, _ := json.Marshal(api.PostUserLoginJSONRequestBody{
						Login:    login,
						Password: password,
					})
					return r
				}(),
			},
			want: want{
				wantStatus: http.StatusInternalServerError,
			},
		},
		{
			name: "Request_With_Failed_Generate_Auth_Token",
			fields: fields{
				userSv: func(ctrl *gomock.Controller) ServiceInterface {
					userSv := mocks.NewMockServiceInterface(ctrl)
					userSv.EXPECT().UserByCredentials(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(&entity.User{}, nil)
					return userSv
				},
				auth: func(ctrl *gomock.Controller) AuthInterface {
					auth := mocks.NewMockAuthInterface(ctrl)
					auth.EXPECT().SetIdentity(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(errors.New("some error"))
					return auth
				},
			},
			args: args{
				r: func() []byte {
					r, _ := json.Marshal(api.PostUserLoginJSONRequestBody{
						Login:    login,
						Password: password,
					})
					return r
				}(),
			},
			want: want{
				wantStatus: http.StatusInternalServerError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			b := handler(ctrl, tt.fields)

			srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				ctx := request.Context()
				request = request.WithContext(
					security.NewContextWithUserIdentity(ctx, entity.NewIdentity(UUID)),
				)
				b.LoginHandler(writer, request)
			}))
			defer srv.Close()

			resp, err := http.Post(srv.URL, "application/json", bytes.NewReader(tt.args.r))
			defer func() {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}()

			if (err != nil) != tt.want.wantErr {
				t.Errorf("LoginHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			assert.Equal(t, tt.want.wantStatus, resp.StatusCode)

			if tt.want.wantAuthHeader && resp.Header.Get("Authorization") == "" {
				t.Errorf("LoginHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
		})
	}
}

func TestAuthHandler_RegisterHandler(t *testing.T) {
	var (
		UUID     = "uuid-1"
		login    = "login"
		password = "password"
	)
	type args struct {
		r []byte
	}
	type want struct {
		wantErr        bool
		wantStatus     int
		wantAuthHeader bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "Request_With_StatusOK",
			fields: fields{
				userSv: func(ctrl *gomock.Controller) ServiceInterface {
					userSv := mocks.NewMockServiceInterface(ctrl)
					userSv.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(&entity.User{}, nil)
					return userSv
				},
				auth: func(ctrl *gomock.Controller) AuthInterface {
					auth := security.NewJWTWrapper("secret")
					return auth
				},
			},
			args: args{
				r: func() []byte {
					r, _ := json.Marshal(api.PostUserLoginJSONRequestBody{
						Login:    login,
						Password: password,
					})
					return r
				}(),
			},
			want: want{
				wantStatus:     http.StatusOK,
				wantAuthHeader: true,
			},
		},
		{
			name: "Request_With_Bad_JSON",
			args: args{
				r: []byte(`{"login":1111,"password"`),
			},
			want: want{
				wantStatus: http.StatusBadRequest,
			},
		},
		{
			name: "Request_Create_User_With_Error",
			fields: fields{
				userSv: func(ctrl *gomock.Controller) ServiceInterface {
					userSv := mocks.NewMockServiceInterface(ctrl)
					userSv.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(nil, errors.New("some error"))
					return userSv
				},
			},
			args: args{
				r: func() []byte {
					r, _ := json.Marshal(api.PostUserLoginJSONRequestBody{
						Login:    login,
						Password: password,
					})
					return r
				}(),
			},
			want: want{
				wantStatus: http.StatusBadRequest,
			},
		},
		{
			name: "Request_With_Failed_Generate_Auth_Token",
			fields: fields{
				userSv: func(ctrl *gomock.Controller) ServiceInterface {
					userSv := mocks.NewMockServiceInterface(ctrl)
					userSv.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(&entity.User{}, nil)
					return userSv
				},
				auth: func(ctrl *gomock.Controller) AuthInterface {
					auth := mocks.NewMockAuthInterface(ctrl)
					auth.EXPECT().SetIdentity(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(errors.New("some error"))
					return auth
				},
			},
			args: args{
				r: func() []byte {
					r, _ := json.Marshal(api.PostUserLoginJSONRequestBody{
						Login:    login,
						Password: password,
					})
					return r
				}(),
			},
			want: want{
				wantStatus: http.StatusInternalServerError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			b := handler(ctrl, tt.fields)

			srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				ctx := request.Context()
				request = request.WithContext(
					security.NewContextWithUserIdentity(ctx, entity.NewIdentity(UUID)),
				)
				b.RegisterHandler(writer, request)
			}))
			defer srv.Close()

			resp, err := http.Post(srv.URL, "application/json", bytes.NewReader(tt.args.r))
			defer func() {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}()

			if (err != nil) != tt.want.wantErr {
				t.Errorf("RegisterHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			assert.Equal(t, tt.want.wantStatus, resp.StatusCode)

			if tt.want.wantAuthHeader && resp.Header.Get("Authorization") == "" {
				t.Errorf("RegisterHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
		})
	}
}

func handler(ctrl *gomock.Controller, fields fields) *AuthHandler {
	var (
		userSv ServiceInterface
		auth   AuthInterface
	)
	if fields.userSv == nil {
		userSv = mocks.NewMockServiceInterface(ctrl)
	} else {
		userSv = fields.userSv(ctrl)
	}
	if fields.auth == nil {
		auth = mocks.NewMockAuthInterface(ctrl)
	} else {
		auth = fields.auth(ctrl)
	}

	return &AuthHandler{
		userSv: userSv,
		auth:   auth,
		logger: log.MockLogger,
	}
}
