/*
Copyright © 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package server contains implementation of REST API server (HTTPServer) for the
// Insights content service. In current version, the following
// REST API endpoints are available:
package server

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-content-service/groups"
)

// HTTPServer in an implementation of Server interface
type HTTPServer struct {
	Config Configuration
	Groups map[string]groups.Group
	Serv   *http.Server
}

// New constructs new implementation of Server interface
func New(config Configuration, groups map[string]groups.Group) *HTTPServer {
	return &HTTPServer{
		Config: config,
		Groups: groups,
	}
}

// Start starts server
func (server *HTTPServer) Start() error {
	address := server.Config.Address
	log.Info().Msgf("Starting HTTP server at '%s'", address)
	router := server.Initialize(address)
	server.Serv = &http.Server{Addr: address, Handler: router}
	var err error

	if server.Config.UseHTTPS {
		err = server.Serv.ListenAndServeTLS("server.crt", "server.key")
	} else {
		err = server.Serv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("Unable to start HTTP/S server")
		return err
	}

	return nil
}

// Stop stops server's execution
func (server *HTTPServer) Stop(ctx context.Context) error {
	return server.Serv.Shutdown(ctx)
}

// Initialize perform the server initialization
func (server *HTTPServer) Initialize(address string) http.Handler {
	log.Info().Msgf("Initializing HTTP server at '%s'", address)

	router := mux.NewRouter().StrictSlash(true)

	if server.Config.EnableCORS {
		log.Info().Msgf("Configuring CORS middleware")
		router.Use(server.addCORSHeaders)
		router.Use(server.handleOptionsMethod)
	}
	server.addEndpointsToRouter(router)
	log.Info().Msgf("Server has been initiliazed")

	return router
}

func (server *HTTPServer) addEndpointsToRouter(router *mux.Router) {
	apiPrefix := server.Config.APIPrefix
	openAPIURL := apiPrefix + filepath.Base(server.Config.APISpecFile)

	// common REST API endpoints
	router.HandleFunc(apiPrefix+MainEndpoint, server.mainEndpoint).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+GroupsEndpoint, server.listOfGroups).Methods(http.MethodGet, http.MethodOptions)

	// OpenAPI specs
	router.HandleFunc(openAPIURL, server.serveAPISpecFile).Methods(http.MethodGet)
}

// addCORSHeaders - middleware for adding headers that should be in any response
func (server *HTTPServer) addCORSHeaders(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			nextHandler.ServeHTTP(w, r)
		})
}

// handleOptionsMethod - middleware for handling OPTIONS method
func (server *HTTPServer) handleOptionsMethod(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
			} else {
				nextHandler.ServeHTTP(w, r)
			}
		})
}
