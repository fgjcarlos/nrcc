package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

func TestShutdownDrainsHTTPBeforeStoppingServicesAndRuntime(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	steps := make(chan string, 3)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		steps <- "handler finished"
		w.WriteHeader(http.StatusNoContent)
	}))
	defer testServer.Close()
	testServer.Config.RegisterOnShutdown(func() { close(release) })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	status := make(chan int, 1)
	go func() {
		res, err := testServer.Client().Do(req)
		if err != nil {
			status <- 0
			return
		}
		_ = res.Body.Close()
		status <- res.StatusCode
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("request did not enter the handler")
	}
	if err := shutdownServer(ctx, testServer.Config, func() { steps <- "services stopped" }, func() { steps <- "runtime stopped" }); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if got := <-status; got != http.StatusNoContent {
		t.Fatalf("in-flight request: status %d, want 204", got)
	}
	close(steps)
	var got []string
	for step := range steps {
		got = append(got, step)
	}
	if want := []string{"handler finished", "services stopped", "runtime stopped"}; !slices.Equal(got, want) {
		t.Fatalf("shutdown order = %v, want %v", got, want)
	}
}
