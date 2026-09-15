package rpcframing

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestWriteThenReadRoundTrips(t *testing.T) {
	var buf bytes.Buffer
	body := []byte(`{"hello":"world"}`)
	if err := WriteMessage(&buf, body); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	got, err := ReadMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("got %q, want %q", got, body)
	}
}

func TestReadMessageMissingContentLength(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("\r\n{}"))
	if _, err := ReadMessage(r); err == nil {
		t.Fatal("expected an error for a missing Content-Length header")
	}
}

func TestReadMessageInvalidContentLength(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("Content-Length: not-a-number\r\n\r\n{}"))
	if _, err := ReadMessage(r); err == nil {
		t.Fatal("expected an error for a non-numeric Content-Length")
	}
}

func TestReadMessageMultipleFramesInSequence(t *testing.T) {
	var buf bytes.Buffer
	WriteMessage(&buf, []byte(`{"a":1}`))
	WriteMessage(&buf, []byte(`{"b":2}`))
	r := bufio.NewReader(&buf)

	first, err := ReadMessage(r)
	if err != nil || string(first) != `{"a":1}` {
		t.Fatalf("first message: got %q, err %v", first, err)
	}
	second, err := ReadMessage(r)
	if err != nil || string(second) != `{"b":2}` {
		t.Fatalf("second message: got %q, err %v", second, err)
	}
}
