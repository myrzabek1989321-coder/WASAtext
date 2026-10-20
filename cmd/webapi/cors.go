package main

import (
	"net/http"

	"github.com/go-chi/cors"
)

// applyCORSHandler configures cross-origin requests for the API
func applyCORSHandler(h http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedHeaders: []string{
			"x-example-header",
			"Content-Type",
			"Authorization",
		},

		AllowedMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
		},

		// Do not modify these values: they are used in evaluation.
		AllowedOrigins: []string{"*"},
		MaxAge:         1,
	})(h)
}
