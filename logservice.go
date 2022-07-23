// Package logservice allows to write data into Elasticsearch
// in an efficient manner, using the BulkIndexer component.
//
// Its main use-case is to ingest data from eg. Vercel log drains
// (https://vercel.com/docs/log-drains#format-and-transport/ndjson-log-drains).
//
// See: https://pkg.go.dev/github.com/elastic/go-elasticsearch/v8/esutil#BulkIndexer
//
package logservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/rs/zerolog"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Service writes data to Elasticsearch.
//
type Service struct {
	indexer     esutil.BulkIndexer
	validSource []string
	logger      zerolog.Logger
}

// Config configures the service.
//
type Config struct {
	ValidSource []string

	IndexName     string
	FlushInterval time.Duration
	FlushBytes    int

	ElasticsearchURL    string
	ElasticsearchAPIKey string

	Logger zerolog.Logger
}

// New returns a new, functional Service.
//
func New(cfg Config) (*Service, error) {
	if cfg.ElasticsearchURL == "" {
		return nil, errors.New("missing Elasticsearch URL")
	}

	if cfg.IndexName == "" {
		return nil, errors.New("missing index name")
	}

	if len(cfg.ValidSource) < 1 {
		cfg.ValidSource = []string{"build", "lambda"}
	}
	sort.Strings(cfg.ValidSource)

	esclient, err := elasticsearch.NewClient(elasticsearch.Config{
		APIKey:        cfg.ElasticsearchAPIKey,
		Addresses:     []string{cfg.ElasticsearchURL},
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    10,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Elasticsearch client: %s", err)
	}

	indexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         cfg.IndexName,
		FlushInterval: cfg.FlushInterval,
		FlushBytes:    cfg.FlushBytes,
		Client:        esclient,
		// NumWorkers:    1,
		OnError: func(_ context.Context, err error) {
			fmt.Println(err)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating the indexer: %s", err)
	}

	return &Service{
		indexer:     indexer,
		validSource: cfg.ValidSource,
		logger:      cfg.Logger,
	}, nil
}

// Write adds b to BulkIndexer.
//
func (s *Service) Write(b []byte) (int, error) {
	var indexerErr error

	if !s.IsValidEntry(b) {
		return 0, nil
	}

	entry, indexerErr := s.TransformEntry(b)
	if indexerErr != nil {
		return 0, indexerErr
	}

	err := s.indexer.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action: "create",
			Body:   bytes.NewReader(entry),

			OnFailure: func(
				ctx context.Context,
				_ esutil.BulkIndexerItem,
				res esutil.BulkIndexerResponseItem,
				err error) {
				if err != nil {
					indexerErr = err
				} else {
					indexerErr = fmt.Errorf("%s: %s", res.Error.Type, res.Error.Reason)
				}
				if indexerErr != nil {
					s.logger.Error().Msgf("Indexing item: %s: %s: %s", res.Error.Cause.Type, res.Error.Cause.Reason, res.Error.Reason)
				}
			},
		},
	)
	if err != nil {
		return 0, err
	}
	return len(b), indexerErr
}

// Flush calls BulkIndexer.Close() to flush the buffers.
//
func (s *Service) Flush(ctx context.Context) error {
	return s.indexer.Close(ctx)
}

// Stats returns BulkIndexer stats.
//
func (s *Service) Stats() esutil.BulkIndexerStats {
	return s.indexer.Stats()
}

// IsValidEntry returns true when the JSON entry is valid for storing.
//
func (s *Service) IsValidEntry(b []byte) bool {
	result := gjson.GetBytes(b, "source").String()
	index := sort.SearchStrings(s.validSource, result)
	return index < len(s.validSource) && s.validSource[index] == result
}

// TransformEntry translates the Vercel JSON into proper structure.
//
func (s *Service) TransformEntry(input []byte) ([]byte, error) {
	var (
		keys []string

		output []byte
		err    error
	)

	output, err = sjson.SetBytes(output, "\\@timestamp", gjson.GetBytes(input, "timestamp").Num)
	if err != nil {
		return output, err
	}

	source := gjson.GetBytes(input, "source").String()
	output, _ = sjson.SetBytes(output, "source", source)
	output, _ = sjson.SetBytes(output, "event.dataset", "vercel")
	output, _ = sjson.SetBytes(output, "data_stream.type", "logs")

	// TODO: https://www.elastic.co/guide/en/ecs/8.2/ecs-service.html
	// service.name
	// service.environment

	switch source {
	case "build":
		keys = []string{"buildId", "deploymentId", "entrypoint", "message", "projectId"}
	case "lambda":
		keys = []string{"deploymentId", "path", "projectId"}
	}

	for _, key := range keys {
		output, err = sjson.SetBytes(output, key, gjson.GetBytes(input, key).String())
		if err != nil {
			return output, err
		}
	}

	if source == "lambda" {
		var message strings.Builder
		message.WriteString("[")
		message.WriteString(gjson.GetBytes(input, "proxy.statusCode").String())
		message.WriteString(" ")
		message.WriteString(http.StatusText(int(gjson.GetBytes(input, "proxy.statusCode").Int())))
		message.WriteString("] ")
		message.WriteString(gjson.GetBytes(input, "proxy.method").String())
		message.WriteString(" ")
		message.WriteString(gjson.GetBytes(input, "proxy.path").String())
		message.WriteString(" ")
		message.WriteString(gjson.GetBytes(input, "message").String())

		output, err = sjson.SetBytes(output, "message", message.String())
		if err != nil {
			return output, err
		}

		output, err = sjson.SetBytes(output, "http.request.method", gjson.GetBytes(input, "proxy.method").String())
		if err != nil {
			return output, err
		}

		output, err = sjson.SetBytes(output, "url.path", gjson.GetBytes(input, "proxy.path").String())
		if err != nil {
			return output, err
		}

		output, err = sjson.SetBytes(output, "http.response.status_code", gjson.GetBytes(input, "proxy.statusCode").String())
		if err != nil {
			return output, err
		}
	}

	// fmt.Printf("output: %s\n", output)
	return output, err
}
