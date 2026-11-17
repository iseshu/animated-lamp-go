package mtproto

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

type limitedWriter struct {
	w      io.Writer
	remain int64
}

func (l *limitedWriter) Write(p []byte) (n int, err error) {
	if l.remain <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.remain {
		p = p[:l.remain]
	}
	n, err = l.w.Write(p)
	l.remain -= int64(n)
	return n, err
}

type Streamer struct {
	client   *telegram.Client
	listener net.Listener
	server   *http.Server
	port     int
	docs     sync.Map // int64 -> *tg.Document
}

func NewStreamer(client *telegram.Client) (*Streamer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind local streamer: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	s := &Streamer{
		client:   client,
		listener: listener,
		port:     port,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/stream/", s.handleStream)

	s.server = &http.Server{
		Handler: mux,
	}

	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("streamer server error: %v", err)
		}
	}()

	log.Printf("Internal MTProto streamer running on http://127.0.0.1:%d", port)
	return s, nil
}

func (s *Streamer) RegisterDocument(doc *tg.Document) string {
	s.docs.Store(doc.ID, doc)
	return fmt.Sprintf("http://127.0.0.1:%d/stream/%d", s.port, doc.ID)
}

func (s *Streamer) handleStream(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	docID, err := strconv.ParseInt(pathParts[2], 10, 64)
	if err != nil {
		http.Error(w, "invalid document ID", http.StatusBadRequest)
		return
	}

	val, ok := s.docs.Load(docID)
	if !ok {
		http.Error(w, "document not found or expired", http.StatusNotFound)
		return
	}

	doc := val.(*tg.Document)
	totalSize := doc.Size

	loc := &tg.InputDocumentFileLocation{
		ID:            doc.ID,
		AccessHash:    doc.AccessHash,
		FileReference: doc.FileReference,
	}

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", doc.MimeType)

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(totalSize, 10))
		w.WriteHeader(http.StatusOK)

		ctx := r.Context()
		_, _ = s.client.Download(loc).Stream(ctx, w)
		return
	}

	// Parse Range: bytes=start-end
	var start, end int64
	rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeStr, "-")

	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 {
		start = 0
	}

	if len(parts) > 1 && parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end >= totalSize {
			end = totalSize - 1
		}
	} else {
		end = totalSize - 1
	}

	contentLength := end - start + 1
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.WriteHeader(http.StatusPartialContent)

	ctx := r.Context()
	lw := &limitedWriter{w: w, remain: contentLength}
	_, _ = s.client.Download(loc).WithOffset(start).Stream(ctx, lw)
}

func (s *Streamer) Close() error {
	return s.server.Close()
}
