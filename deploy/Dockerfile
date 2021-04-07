FROM golang:1.16.3-alpine AS src

# Install git
RUN set -ex; \
    apk update; \
    apk add --no-cache git

# Copy Repository
WORKDIR /go/src/github.com/tamasbrandstadter/payments-api/
COPY . ./

# Build Go Binary
RUN set -ex; \
    CGO_ENABLED=0 GOOS=linux go build -o ./api ./cmd/api;

# Final image, no source code
FROM alpine:latest

# Install Root Ceritifcates
RUN set -ex; \
    apk update; \
    apk add --no-cache \
     ca-certificates

WORKDIR /opt/
COPY --from=src /go/src/github.com/tamasbrandstadter/payments-api/api .

# Run Go Binary
CMD /opt/api