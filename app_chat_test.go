package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAnthropicChatRetriesEarlyEOF(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(body), `"stream":true`) ||
			r.URL.Path != "/anthropic/v1/messages" || r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("invalid request: path=%q, body=%q, read error=%v", r.URL.Path, body, err)
		}
		if attempts.Add(1) == 1 {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			conn.Close()
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"message_start\",\"message\":{\"role\":\"assistant\"}}\n\n"+
			"data: {\"type\":\"content_block_start\",\"content_block\":{\"type\":\"text\",\"text\":\"ok\"}}\n\n"+
			"data: {\"type\":\"content_block_stop\"}\n\n"+
			"data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer server.Close()

	app := NewApp("")
	result, err := app.ChatCompletion("test-key", server.URL+"/anthropic", "test-model", `{"messages":[]}`, "anthropic", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, `"text":"ok"`) || attempts.Load() != 2 {
		t.Fatalf("result = %s, attempts = %d", result, attempts.Load())
	}
}
