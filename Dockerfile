# syntax=docker/dockerfile:1.4
FROM golang:1.18-alpine3.15 AS builder

WORKDIR /app

RUN set -eux; \
	\
	apk add --no-cache --virtual .deps \
		ca-certificates \
		git \
	;

RUN apk update && apk add --no-cache --quiet ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -tags netgo -o /app/server ./cmd/server/main.go

RUN apk del --no-network .deps;

FROM alpine3.15

RUN apk update && apk add --no-cache --quiet ca-certificates curl


RUN addgroup -g 10001 -S nonroot && adduser -u 10000 -S -G nonroot -h /home/nonroot nonroot

RUN ln -sf /dev/stdout /var/log/app.log
RUN useradd -r -u 1001 -g root nonroot
RUN chmod -R g+rwX /var/log

USER nonroot

COPY --from=builder /app/server /server

ENV PORT 8080
EXPOSE $PORT
ENTRYPOINT ["/server"]
