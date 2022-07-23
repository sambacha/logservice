FROM golang:1-alpine AS Builder

WORKDIR /app

RUN apk update && apk add --no-cache --quiet ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -tags netgo -o /app/server ./cmd/server/main.go

FROM alpine

RUN apk update && apk add --no-cache --quiet ca-certificates curl

COPY --from=Builder /app/server /server

ENV PORT 8080
EXPOSE $PORT
ENTRYPOINT ["/server"]
