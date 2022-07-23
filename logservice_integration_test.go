//go:build integration
// +build integration

package logservice_test

import (
	"context"
	"embed"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	logservice "github.com/cesko-digital/vercel-logging"
)

//go:embed *.ndjson
var fs embed.FS

func TestService(t *testing.T) {
	svc, err := logservice.New(logservice.Config{
		IndexName:           "logservice-logs",
		ElasticsearchURL:    os.Getenv("ELASTICSEARCH_URL"),
		ElasticsearchAPIKey: os.Getenv("ELASTICSEARCH_API_KEY"),

		FlushInterval: 50 * time.Second,
		FlushBytes:    1e+6,

		Logger: log.Output(zerolog.ConsoleWriter{Out: os.Stderr}),
	})
	if err != nil {
		t.Fatalf("Error creating service: %s", err)
	}

	server := httptest.NewServer(svc)
	defer server.Close()

	t.Run("Multiline", func(t *testing.T) {
		reqBody, _ := fs.Open("testdata.ndjson")

		req := httptest.NewRequest("POST", "http://example.com/", reqBody)
		w := httptest.NewRecorder()

		svc.ServeHTTP(w, req)
		svc.Flush(context.Background())

		res := w.Result()
		body, _ := io.ReadAll(res.Body)

		if res.StatusCode != 200 {
			t.Errorf("Unexpected status code: %d", res.StatusCode)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Errorf("Error decoding response body: %s", err)
		}
		if result["error"] != false {
			t.Errorf("Unexpected body: %s", body)
		}

		stats := svc.Stats()
		t.Logf("Service stats: %+v", stats)

		fieldNames := []string{"NumAdded", "NumFlushed", "NumCreated"}
		for _, f := range fieldNames {
			v := reflect.ValueOf(stats).FieldByName(f)
			if v.Uint() != 3 {
				t.Errorf("Unexpected value for [%s]: %d", f, v.Uint())
			}
		}
	})
}
