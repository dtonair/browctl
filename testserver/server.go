package testserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

type Server struct {
	*httptest.Server
}

func New() *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<!doctype html>
<html><body>
<form id="login">
  <input name="email" aria-label="Email">
  <button id="submit" type="button" onclick="document.body.dataset.clicked='yes'; location.href='/dashboard'">Sign in</button>
</form>
<div id="status">Ready</div>
</body></html>`)
	})
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<!doctype html><html><body><h1 id="title">Dashboard</h1><p class="account">Account OK</p></body></html>`)
	})
	mux.HandleFunc("/dupes", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<!doctype html><html><body><button>Same</button><button>Same</button></body></html>`)
	})
	return &Server{Server: httptest.NewServer(mux)}
}
