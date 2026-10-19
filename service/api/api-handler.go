package api

import (
	"net/http"
)

// Handler returns an instance of chi.Mux that handles the APIs registered here
func (rt *_router) Handler() http.Handler {
	// Register routes
	rt.router.Get("/", rt.getHelloWorld)

	// Example of a route with its own middleware, attached with With(...)
	rt.router.With(rt.exampleMiddleware).Get("/context", rt.getContextReply)

	// Example data routes (read and write through the database)
	rt.router.Get("/name", rt.getName)
	rt.router.Put("/name", rt.setName)

	// Special routes
	rt.router.Get("/liveness", rt.liveness)

	// New API to doLogin, Users, Photo, Conversations
	rt.router.Post("/session", rt.doLogin)
	rt.router.Get("/users", rt.getUsers)
	rt.router.Put("/users/me/username", rt.setMyUserName)
	rt.router.Put("/users/me/photo", rt.setMyPhoto)
	rt.router.Post("/conversations", rt.createConversation)
	rt.router.Get("/conversations", rt.getMyConversations)

	return rt.router
}
