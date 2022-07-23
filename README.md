# [`ldrain`](#)

## Abstract

`ldrain` enables logs to write data into Elasticsearch in an efficient manner, using the [BulkIndexer component](https://pkg.go.dev/github.com/elastic/go-elasticsearch/v8/esutil#BulkIndexer).

Its main use-case is to ingest data from platforms such as [Vercel via their log drains](https://vercel.com/docs/log-drains#format-and-transport/ndjson-log-drains).

### Build

```console
usr:~ $ docker-compose up --build
```

### Run

```bash
ELASTICSEARCH_URL=http://localhost:9200 go run bin/setup/main.go
```

### Usage

```bash
for i in {1..100}; do 
  curl -X POST localhost:8080 --data-binary $'{"@timestamp":"2022-07-20","message":"assert invariant1"}'
done
```


```hurl
[Asserts]
jsonpath "$.date1" matches "\\d{4}-\\d{2}-\\d{2}"
jsonpath "$.date1" matches /\d{4}-\d{2}-\d{2}/
jsonpath "$.date2" matches /\d{4}-\d{2}-\d{2}/
jsonpath "$.date1" matches /^\d{4}-\d{2}-\d{2}$/
jsonpath "$.date2" not matches /^\d{4}-\d{2}-\d{2}$/
jsonpath "$.path1" matches /aa\/bb/
jsonpath "$.path2" matches /aa\\bb/
