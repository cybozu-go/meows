// fake-slack is a minimal Slack API server for kindtest.
// It records messages posted by chat.postMessage.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/slack-go/slack"
)

// Message is a message posted by chat.postMessage.
type Message struct {
	Channel     string             `json:"channel"`
	Text        string             `json:"text"`
	ThreadTS    string             `json:"thread_ts"`
	TS          string             `json:"ts"`
	Attachments []slack.Attachment `json:"attachments"`
}

type server struct {
	mu       sync.Mutex
	messages []Message
}

// handleConnectionsOpen returns a non-fatal error so that the socketmode client keeps retrying
// without exiting slack-agent. kindtest does not test Socket Mode.
func handleConnectionsOpen(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"ok": false, "error": "socket_mode_not_supported"})
}

func (s *server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	msg := Message{
		Channel:  r.PostForm.Get("channel"),
		Text:     r.PostForm.Get("text"),
		ThreadTS: r.PostForm.Get("thread_ts"),
	}
	if a := r.PostForm.Get("attachments"); a != "" {
		if err := json.Unmarshal([]byte(a), &msg.Attachments); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid_attachments"})
			return
		}
	}

	s.mu.Lock()
	msg.TS = fmt.Sprintf("%d.%06d", time.Now().Unix(), len(s.messages))
	s.messages = append(s.messages, msg)
	s.mu.Unlock()

	log.Printf("received message: channel=%s ts=%s", msg.Channel, msg.TS)
	writeJSON(w, map[string]any{
		"ok":      true,
		"channel": msg.Channel,
		"ts":      msg.TS,
	})
}

func (s *server) handleMessages(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, s.messages)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func main() {
	addr := flag.String("listen-addr", ":8080", "The address to listen on")
	flag.Parse()

	s := &server{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/apps.connections.open", handleConnectionsOpen)
	mux.HandleFunc("POST /api/chat.postMessage", s.handlePostMessage)
	mux.HandleFunc("GET /messages", s.handleMessages)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}
