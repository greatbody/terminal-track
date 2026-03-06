package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/greatbody/terminal-track/internal/db"
)

//go:embed templates/*
var templateFS embed.FS

// Server serves the web timeline UI.
type Server struct {
	db   *db.DB
	tmpl *template.Template
}

// New creates a new web server.
func New(d *db.DB) (*Server, error) {
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Local().Format("15:04:05")
		},
		"formatDate": func(t time.Time) string {
			return t.Local().Format("2006-01-02")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Local().Format("2006-01-02 15:04:05")
		},
		"exitClass": func(code *int) string {
			if code == nil {
				return "exit-unknown"
			}
			if *code == 0 {
				return "exit-ok"
			}
			return "exit-fail"
		},
		"exitText": func(code *int) string {
			if code == nil {
				return ""
			}
			if *code == 0 {
				return "0"
			}
			return fmt.Sprintf("%d", *code)
		},
		"terminalLabel": func(r db.Record) string {
			label := r.Terminal
			if label == "" || label == "unknown" {
				label = "terminal"
			}
			if r.TmuxPane != "" {
				label = "tmux " + r.TmuxPane
			}
			return label
		},
		"shortTTY": func(tty string) string {
			// "/dev/ttys003" -> "ttys003"
			if idx := strings.LastIndex(tty, "/"); idx >= 0 {
				return tty[idx+1:]
			}
			return tty
		},
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Server{db: d, tmpl: tmpl}, nil
}

// Handler returns the HTTP handler for the web server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/commands", s.handleAPI)
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	opts := s.parseQueryOpts(r)
	opts.Limit = 100

	records, err := s.db.Query(opts)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	total, _ := s.db.Count(opts)

	data := struct {
		Records []db.Record
		Total   int
		Search  string
		Dir     string
		Session string
	}{
		Records: records,
		Total:   total,
		Search:  r.URL.Query().Get("q"),
		Dir:     r.URL.Query().Get("dir"),
		Session: r.URL.Query().Get("session"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	opts := s.parseQueryOpts(r)
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	records, err := s.db.Query(opts)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	total, _ := s.db.Count(opts)

	resp := struct {
		Records []db.Record `json:"records"`
		Total   int         `json:"total"`
	}{
		Records: records,
		Total:   total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) parseQueryOpts(r *http.Request) db.QueryOptions {
	q := r.URL.Query()
	opts := db.QueryOptions{
		Search:    q.Get("q"),
		Directory: q.Get("dir"),
		Session:   q.Get("session"),
	}

	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 {
		opts.Limit = n
	}
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n > 0 {
		opts.Offset = n
	}

	if since := q.Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			opts.Since = &t
		}
	}
	if until := q.Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			opts.Until = &t
		}
	}

	return opts
}
