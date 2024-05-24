ARG GOLANG_VERSION=1.22
ARG ALPINE_VERSION=3.19

###
# BUILD
###
FROM golang:${GOLANG_VERSION}-alpine${ALPINE_VERSION} as builder

WORKDIR /app

# copying sources
COPY go.mod go.sum  *.go ./
# compiling
RUN go mod download
RUN go build -o /integration-toolbox-webserver

#####
# Application
#####
FROM alpine:${ALPINE_VERSION}

WORKDIR /

# bin
COPY --from=builder /integration-toolbox-webserver /usr/local/bin/integration-toolbox-webserver
RUN apk add --no-cache curl libcap && \
    setcap "cap_net_raw=+ep" /usr/local/bin/integration-toolbox-webserver

# UI
COPY ui /ui
RUN chown -R nobody:nobody /ui

EXPOSE 8080
USER nobody:nobody
ENTRYPOINT ["/usr/local/bin/integration-toolbox-webserver"]
