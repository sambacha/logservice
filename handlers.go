package logservice

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
)

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.RequestURI() != "/" {
		handleError("Not found", http.StatusNotFound)(w, r)
		return
	}

	switch r.Method {
	case "HEAD", "GET":
		handlePing()(w, r)
	case "POST":
		handleIngest(s)(w, r)
	default:
		handleError("Unsupported method", http.StatusMethodNotAllowed)(w, r)
		return
	}
}

func handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	}
}

func handleError(msg string, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Server error:", msg)
		w.WriteHeader(status)
		if r.Method != "HEAD" {
			fmt.Fprintln(w, msg)
		}
	}
}

func handleIngest(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		scanner := bufio.NewScanner(r.Body)
		for scanner.Scan() {
			_, err := svc.Write(scanner.Bytes())
			if err != nil {
				fmt.Fprintln(os.Stderr, "ingest: write:", err)
				handleError(`{"error":true}`, http.StatusInternalServerError)(w, r)
				return
			}
		}
		if err = scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "ingest: read:", err)
			handleError(`{"error":true}`, http.StatusInternalServerError)(w, r)
			return
		}

		fmt.Fprintln(w, `{"error":false}`)
	}
}
