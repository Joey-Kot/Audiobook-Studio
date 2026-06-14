package tts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSynthesizeSuccess(t *testing.T) {
	client := New("https://example.test/speech", "token", "model", 5)
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("missing auth header")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["model"] != "model" || body["input"] != "hello" || body["voice"] != "alloy" {
			t.Fatalf("unexpected body %#v", body)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(bytes.NewBufferString("audio")),
			Header:     make(http.Header),
		}, nil
	})}

	got, err := client.Synthesize("hello", `{"voice":"alloy"}`)
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if string(got) != "audio" {
		t.Fatalf("unexpected audio %q", got)
	}
}

func TestSynthesizeErrorStatus(t *testing.T) {
	client := New("https://example.test/speech", "", "model", 5)
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Status:     "502 Bad Gateway",
			Body:       io.NopCloser(bytes.NewBufferString("nope")),
			Header:     make(http.Header),
		}, nil
	})}
	if _, err := client.Synthesize("hello", `{"voice":"alloy"}`); err == nil {
		t.Fatal("expected status error")
	}
}

func TestSynthesizeTimeout(t *testing.T) {
	client := New("https://example.test/speech", "", "model", 1)
	client.HTTPClient = &http.Client{
		Timeout: 1 * time.Millisecond,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			select {
			case <-time.After(50 * time.Millisecond):
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(bytes.NewBufferString("late")),
					Header:     make(http.Header),
				}, nil
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
		}),
	}
	if _, err := client.Synthesize("hello", `{"voice":"alloy"}`); err == nil {
		t.Fatal("expected timeout")
	} else if timeoutErr := (net.Error)(nil); !errors.As(err, &timeoutErr) || !timeoutErr.Timeout() {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}
