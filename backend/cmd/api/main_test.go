package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/config"
)

type fakeHTTPServer struct {
	listen      chan struct{}
	shutdown    chan struct{}
	shutdownErr error
}

func (f *fakeHTTPServer) ListenAndServe() error {
	<-f.listen
	return http.ErrServerClosed
}

func (f *fakeHTTPServer) Shutdown(context.Context) error {
	close(f.shutdown)
	close(f.listen)
	return f.shutdownErr
}

func TestServeGracefullyShutsDownOnContextCancellation(t *testing.T) {
	server := &fakeHTTPServer{listen: make(chan struct{}), shutdown: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)
	go func() {
		result <- serve(ctx, &config.Config{AppEnv: "test", AppVersion: "1.2.3", Port: "8080"}, server, testLogger())
	}()

	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("serve() error = %v", err)
		}
	case <-server.shutdown:
		if err := <-result; err != nil {
			t.Fatalf("serve() error = %v", err)
		}
	}
}

func TestServeReturnsListenFailure(t *testing.T) {
	wantErr := errors.New("listen failed")
	server := &listenErrorServer{err: wantErr}

	err := serve(context.Background(), &config.Config{AppEnv: "test", AppVersion: "1.2.3", Port: "8080"}, server, testLogger())
	if !strings.Contains(err.Error(), "serve API: listen failed") {
		t.Fatalf("serve() error = %v", err)
	}
}

type listenErrorServer struct {
	err error
}

func (s *listenErrorServer) ListenAndServe() error { return s.err }

func (*listenErrorServer) Shutdown(context.Context) error { return nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}
