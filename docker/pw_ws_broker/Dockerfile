FROM golang:1.20 AS build

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOAMD64=v3

ARG IMAGE_VERSION=dev

WORKDIR /app

# prefill the build steps
COPY go.mod /app
COPY go.sum /app
RUN go mod verify && go mod download

# Copy the code into the container
COPY . /app

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Build the application
RUN set -x && go build -buildvcs=false -ldflags="-X main.version=${IMAGE_VERSION//\"/}" plane.watch/cmd/pw_ws_broker

# test binary (make sure it will run on this arch)
RUN ./pw_ws_broker --help

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/pw_ws_broker /app
EXPOSE 80 443

HEALTHCHECK --interval=15s --timeout=3s CMD wget --spider http://localhost:9603/status

# Command to run
CMD [ "/app/pw_ws_broker", "daemon" ]
