```
docker-compose up --build
```

```
ELASTICSEARCH_URL=http://localhost:9200 go run bin/setup/main.go
```

```
for i in {1..100}; do 
  curl -X POST localhost:8080 --data-binary $'{"@timestamp":"2022-04-01","message":"Test 1"}'
done
```
