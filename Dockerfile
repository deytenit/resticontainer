ARG RESTIC_VERSION=latest
FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o resticontainer ./cmd/resticontainer

FROM restic/restic:${RESTIC_VERSION}
COPY --from=builder /app/resticontainer /usr/local/bin/resticontainer
ENTRYPOINT ["/usr/local/bin/resticontainer"]
