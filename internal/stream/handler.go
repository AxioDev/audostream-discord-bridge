package stream

import (
	"net/http"

	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

type Handler struct {
	hub          *Hub
	sampleRate   uint32
	channelCount uint16
	bufferSize   int
}

func NewHandler(hub *Hub, sampleRate uint32, channelCount uint16, bufferSize int) *Handler {
	return &Handler{
		hub:          hub,
		sampleRate:   sampleRate,
		channelCount: channelCount,
		bufferSize:   bufferSize,
	}
}

type flushWriter struct {
	http.ResponseWriter
	flusher http.Flusher
}

func (w *flushWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if err != nil {
		return n, err
	}
	w.flusher.Flush()
	return n, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "audio/ogg")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := h.hub.Register(h.bufferSize)
	defer h.hub.Unregister(client.ID())

	writer, err := oggwriter.NewWith(&flushWriter{ResponseWriter: w, flusher: flusher}, h.sampleRate, h.channelCount)
	if err != nil {
		http.Error(w, "Failed to start audio stream", http.StatusInternalServerError)
		return
	}
	defer writer.Close()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case packet, ok := <-client.Packets():
			if !ok {
				return
			}
			if err := writer.WriteRTP(packet); err != nil {
				return
			}
		}
	}
}
