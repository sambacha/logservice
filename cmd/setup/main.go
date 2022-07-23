package main

import (
	"embed"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

//go:embed *.json
var fs embed.FS

var (
	indexName string
)

func init() {
	flag.StringVar(&indexName, "index", "logservice-logs", "Elasticsearch index name")
	flag.Parse()
}

func main() {
	log.SetFlags(0)

	esclient, e := elasticsearch.NewClient(elasticsearch.Config{
		APIKey:    os.Getenv("ELASTICSEARCH_API_KEY"),
		Addresses: []string{os.Getenv("ELASTICSEARCH_URL")},

		Logger: &elastictransport.ColorLogger{Output: os.Stdout, EnableResponseBody: true},
	})
	if e != nil {
		log.Fatalf("Error creating Elasticsearch client: %s", e)
	}

	// log.Println(esclient.Info())

	// Create a data stream
	// See: https://www.elastic.co/guide/en/elasticsearch/reference/current/set-up-a-data-stream.html

	var (
		res *esapi.Response
		err error
	)

	dataPolicy, _ := fs.Open("ilm-policy.json")
	res, err = esclient.ILM.PutLifecycle("logservice", esclient.ILM.PutLifecycle.WithBody(dataPolicy))
	handleError("creating lifecycle", err, res)

	dataSettings, _ := fs.Open("settings.json")
	res, err = esclient.Cluster.PutComponentTemplate("logservice-settings", dataSettings)
	handleError("creating settings", err, res)

	dataMappings, _ := fs.Open("mappings.json")
	res, err = esclient.Cluster.PutComponentTemplate("logservice-mappings", dataMappings)
	handleError("creating mappings", err, res)

	res, err = esclient.Indices.PutIndexTemplate("logservice",
		strings.NewReader(
			`{
			"index_patterns": ["`+indexName+`*"],
			"data_stream": { },
			"composed_of": [ "logservice-settings", "logservice-mappings" ]
		}`),
	)
	handleError("creating index template", err, res)

	res, err = esclient.Indices.GetDataStream(esclient.Indices.GetDataStream.WithName(indexName))
	if err != nil {
		handleError("data stream: %s", err, nil)
	}
	if res.StatusCode == 404 {
		res, err = esclient.Indices.CreateDataStream(indexName)
		handleError("creating data stream", err, res)
	}
}

func handleError(msg string, err error, res *esapi.Response) {
	if err != nil {
		log.Fatalf("Error: %s: %s", msg, err)
	}
	if res.IsError() {
		log.Fatalf("Error: %s: %s", msg, res)
	}
}
