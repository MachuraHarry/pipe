package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/harry/pipe/pkg/eval"
	"github.com/harry/pipe/pkg/lexer"
	"github.com/harry/pipe/pkg/object"
	"github.com/harry/pipe/pkg/parser"
)

var outputBuf strings.Builder

func init() {
	object.PrintHook = func(args ...object.Object) {
		for i, arg := range args {
			if i > 0 {
				outputBuf.WriteByte(' ')
			}
			outputBuf.WriteString(arg.Inspect())
		}
		outputBuf.WriteByte('\n')
	}
}

func runCode(code string) string {
	outputBuf.Reset()

	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	if errs := p.Errors(); len(errs) > 0 {
		result := "Parse errors:\n"
		for _, e := range errs {
			result += "  " + e + "\n"
		}
		return result
	}

	env := object.NewEnvironment()
	ctx := eval.NewEvalContext("<api>")
	result := ctx.Eval(program, env)
	if result != nil && result.Type() == object.ERROR {
		outputBuf.WriteString("Error: " + result.Inspect() + "\n")
	}
	return outputBuf.String()
}

type RunRequest struct {
	Code string `json:"code"`
}

type RunResponse struct {
	Output string `json:"output"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "use POST"})
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	if req.Code == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing 'code'"})
		return
	}

	if len(req.Code) > 50000 {
		writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{Error: "code exceeds 50,000 chars"})
		return
	}

	output := runCode(req.Code)
	writeJSON(w, http.StatusOK, RunResponse{Output: output})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/run", runHandler)
	mux.HandleFunc("/health", healthHandler)

	addr := ":3001"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      withLogging(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	fmt.Printf("pipe-api listening on %s\n", addr)
	log.Fatal(server.ListenAndServe())
}
