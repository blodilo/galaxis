package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"galaxis/internal/jobs"

	"github.com/go-chi/chi/v5"
)

// getGenerateProgress handles GET /api/v1/generate/{jobID}/progress.
// Streams Server-Sent Events with incremental progress for one generation job.
// Supports reconnect via Last-Event-ID header: replays all events with Seq > lastID.
func getGenerateProgress(store *jobs.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "jobID")

		// Parse Last-Event-ID for reconnect replay.
		afterSeq := -1
		if leid := r.Header.Get("Last-Event-ID"); leid != "" {
			if n, err := strconv.Atoi(leid); err == nil {
				afterSeq = n
			}
		}

		replay, live, ok := store.Subscribe(jobID, afterSeq)
		if !ok {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
		w.WriteHeader(http.StatusOK)

		flusher, canFlush := w.(http.Flusher)
		if !canFlush {
			return
		}

		// middleware.Timeout(0) creates an immediately-expiring context which would
		// kill the stream right after replay. Build a deadline-free context that still
		// cancels when the client disconnects (r.Context() signals that).
		sseCtx, sseCancel := context.WithCancel(context.Background())
		defer sseCancel()
		go func() {
			select {
			case <-r.Context().Done():
				sseCancel()
			case <-sseCtx.Done():
			}
		}()

		lastSent := afterSeq
		writeEv := func(ev jobs.ProgressEvent) {
			data, _ := json.Marshal(ev)
			_, _ = fmt.Fprintf(w, "id: %d\ndata: %s\n\n", ev.Seq, data)
			flusher.Flush()
			lastSent = ev.Seq
		}

		// Send replay events (history since last disconnect).
		for _, ev := range replay {
			writeEv(ev)
		}

		// Stream live events until job done or client disconnects.
		for {
			select {
			case ev, open := <-live:
				if !open {
					// Channel closed = job done or error.
					_, _ = fmt.Fprintf(w, "event: done\ndata: {}\n\n")
					flusher.Flush()
					return
				}
				// Skip events already sent via replay (channel may buffer pre-replay events).
				if ev.Seq > lastSent {
					writeEv(ev)
				}
			case <-sseCtx.Done():
				return
			}
		}
	}
}
