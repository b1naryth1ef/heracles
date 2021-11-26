FROM golang:1.16-alpine AS builder

RUN mkdir -p /usr/src/app
WORKDIR /usr/src/app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN go build -v -o /bin/heracles

FROM alpine:latest
COPY --from=builder /bin/heracles /bin/heracles

ENTRYPOINT ["/bin/heracles"]

