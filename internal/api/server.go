package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

// NewServerHandler Создаёт обработчик.
func NewServerHandler(srv ServerInterface, router *mux.Router, middleware []MiddlewareFunc) (http.Handler, error) {
	handler := HandlerWithOptions(
		srv,
		GorillaServerOptions{
			BaseURL:     "/api",
			BaseRouter:  router.Name("api").Subrouter(),
			Middlewares: middleware,
		})

	return handler, nil
}
