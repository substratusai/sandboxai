package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	v1 "github.com/substratusai/sandboxai/api/v1"
	"github.com/substratusai/sandboxai/sandboxaid/client"

	stdlog "log"
)

type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

func SetLogger(logger Logger) {
	log = logger
}

var log Logger = stdlog.New(os.Stderr, "", stdlog.LstdFlags)

type Handler struct {
	http.Handler
	client client.Client
}

func NewHandler(client client.Client) *Handler {
	r := chi.NewRouter()

	h := &Handler{
		Handler: r,
		client:  client,
	}

	// Log to stderr.
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
		Logger: log,
	}))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		r.Route("/spaces/{space}/sandboxes", func(r chi.Router) {
			r.Post("/", h.v1PostSandbox)
		})
		r.Route("/spaces/{space}/sandboxes/{name}", func(r chi.Router) {
			r.Get("/", h.v1GetSandbox)
			r.Delete("/", h.v1DeleteSandbox)
			r.Post("/tools:*", h.v1ProxyToSandbox)
		})
	})
	return h
}

func (h *Handler) v1PostSandbox(w http.ResponseWriter, r *http.Request) {
	space := chi.URLParam(r, "space")

	if space != "default" {
		sendUnimplementedSpaceError(w, r, space)
		return
	}

	var s v1.CreateSandboxRequest
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		sendError(w, r, err, http.StatusBadRequest)
		return
	}

	created, err := h.client.CreateSandbox(r.Context(), space, &s)
	if err != nil {
		sendError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&created.Sandbox); err != nil {
		sendError(w, r, err, http.StatusInternalServerError)
		return
	}
}

func (h *Handler) v1GetSandbox(w http.ResponseWriter, r *http.Request) {
	space := chi.URLParam(r, "space")
	name := chi.URLParam(r, "name")

	if space != "default" {
		sendUnimplementedSpaceError(w, r, space)
		return
	}

	s, err := h.client.GetSandbox(r.Context(), space, name)
	if err != nil {
		if errors.Is(err, client.ErrSandboxNotFound) {
			sendError(w, r, err, http.StatusNotFound)
			return
		}
		sendError(w, r, err, http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(&s.Sandbox); err != nil {
		sendError(w, r, err, http.StatusInternalServerError)
		return
	}
}

func (h *Handler) v1DeleteSandbox(w http.ResponseWriter, r *http.Request) {
	space := chi.URLParam(r, "space")
	name := chi.URLParam(r, "name")

	if space != "default" {
		sendUnimplementedSpaceError(w, r, space)
		return
	}

	if err := h.client.DeleteSandbox(r.Context(), space, name); err != nil {
		if errors.Is(err, client.ErrSandboxNotFound) {
			sendError(w, r, err, http.StatusNotFound)
			return
		}
		sendError(w, r, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) v1ProxyToSandbox(w http.ResponseWriter, r *http.Request) {
	space := chi.URLParam(r, "space")
	name := chi.URLParam(r, "name")

	if space != "default" {
		sendUnimplementedSpaceError(w, r, space)
		return
	}

	s, err := h.client.GetSandbox(r.Context(), space, name)
	if err != nil {
		sendError(w, r, err, http.StatusNotFound)
		return
	}

	containerURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", s.BoxHostPort))
	if err != nil {
		sendError(w, r, err, http.StatusInternalServerError)
		return
	}
	r.URL.Path = strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/v1/spaces/%s/sandboxes/%s", space, name))
	proxy := httputil.NewSingleHostReverseProxy(containerURL)
	proxy.ServeHTTP(w, r)
}

func sendUnimplementedSpaceError(w http.ResponseWriter, r *http.Request, space string) {
	sendError(w, r, fmt.Errorf("space %q not found: current only the %q is supported", space, "default"), http.StatusNotFound)
}

func sendError(w http.ResponseWriter, r *http.Request, err error, status int) {
	w.WriteHeader(status)
	if status >= 500 {
		log.Printf("error serving request: %s: %v", r.URL.Path, err)
		err = fmt.Errorf("%v", http.StatusText(status))
	}
	json.NewEncoder(w).Encode(v1.Error{Message: err.Error()})
}
