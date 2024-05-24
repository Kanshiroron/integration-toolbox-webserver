<p align="center">
  <img alt="Integration Toolbox WebServer" title="Integration Toolbox WebServer" src="ui/icon.png" style="width:128px;height:128px;">
</p>

# Integration Toolbox WebServer

[![Docker Hub repo](https://img.shields.io/docker/pulls/kanshiroron/integration-toolbox-webserver.svg)](https://hub.docker.com/r/kanshiroron/integration-toolbox-webserver)

The Integration Toolbox WebServer is a webserver which exposes a set of endpoints that makes the testing of integrations easier. It can be helpful to debug proxies, reverse proxies, latencies, CORS issues, HTTP requests, network connectivity, database connections and many more.

Most endpoints are easily reachable with simple curl command but if you feel lazy a web UI is also available (which contains most endpoints).

:warning: Even though this server offers basic auth and TLS capabilities, it **SHOULD NOT** be accessible from the Internet.

**Table of Contents**

- [Integration Toolbox WebServer](#integration-toolbox-webserver)
  - [Run](#run)
    - [Docker Image](#docker-image)
    - [From Source](#from-source)
    - [Kubernetes](#kubernetes)
  - [Configuration](#configuration)
    - [General](#general)
    - [Basic Auth](#basic-auth)
    - [TLS](#tls)
    - [Monitoring](#monitoring)
  - [Endpoints](#endpoints)
    - [`/crash`](#crash)
    - [`/download`](#download)
    - [`/echo`](#echo)
    - [`/echo/form`](#echoform)
    - [`/echo/raw`](#echoraw)
    - [`/ping`](#ping)
    - [`/request`](#request)
    - [`/sleep`](#sleep)
    - [`/tcp`](#tcp)
    - [`/status_code`](#status_code)
    - [`POST /database/connect`](#post-databaseconnect)
    - [`POST /database/query`](#post-databasequery)
    - [`/cpu/load`](#cpuload)
    - [`/cpu/reset`](#cpureset)
    - [`/ram/increase`](#ramincrease)
    - [`/ram/decrease`](#ramdecrease)
    - [`/ram/leak`](#ramleak)
    - [`/ram/reset`](#ramreset)
    - [`/ram/status`](#ramstatus)
    - [`/static/`](#static)
    - [`/ui/`](#ui)
    - [`/started`](#started)
      - [GET `/started`](#get-started)
      - [POST `/started`](#post-started)
    - [`/alive`](#alive)
    - [`/ready`](#ready)
  - [Contribution](#contribution)
  - [License](#license)
  - [Credits](#credits)
    - [Authors](#authors)
    - [Dependencies](#dependencies)
    - [Other](#other)

## Run

### Docker Image

Running the server using the official docker image is the easiest way. Just run:

```bash
docker run -d --name itw -p 127.0.0.1:8080:8080 --cap-add NET_RAW kanshiroron/integration-toolbox-webserver
```

The `--add-cap NET_RAW` is neede for the `/ping` endpoint to work (since the server send "unprivileged" UDP packets). If you do not plan on using this endpoint, you can safely remove this flad.

Once the container is running, you can access the web interface from your browser at the URL: [http://localhost:8080/ui/](http://localhost:8080/ui/).

You can also run the Docker image with the `--read-only` option, as long as the program has write permissions for the temp folder (see the [configuration](#configuration) section, defaults to `/tmp/integration-toolbox-webserver`).

Just in case, the docker image is shipped with the `curl` command, to send requests directly from the container.

**More examples:**

```bash
docker run -d --name itw -p 127.0.0.1:8080:8080 --cap-add NET_RAW -e DEBUG=true kanshiroron/integration-toolbox-webserver # to enable debug logs
docker run -d --name itw -p 127.0.0.1:8080:8080 --cap-add NET_RAW -v /path/to/static/folder:/static -e STATIC_FOLDER=/static kanshiroron/integration-toolbox-webserver # serves a static folder
docker run -d --name itw -p 127.0.0.1:8080:8080 --cap-add NET_RAW -v /path/to/temp/folder:/tmp/integration-toolbox-webserver --read-only kanshiroron/integration-toolbox-webserver # read only (make sure user `65534:65534`) has the right to write under the mounted temp folder.
```

### From Source

**Prerequisites**

- [GoLang 1.22+](https://go.dev/doc/install)
- Make (optional)

To run the server directly from the source code, follow these steps: clone this repo, then open a terminal and run:

```bash
cd /path/to/project
go mod download # to download dependencies
go run . # or "make run"
```

On a Linux OS, the `/ping` endpoint will not work out of the box, due to some missing privileges. Indeed, the endpoint sends "unprivileged" UDP packets which are blocked by default. To add such privileges you will need to first compile the webserver and then call `setcap`:

```bash
cd /path/to/project
go build -o /path/to/binary # compiles the project
setcap cap_net_raw=+ep /path/to/binary # add capabilities to the compiled binary
```

Once the server is running, you can access the UI from your browser at the URL: [http://localhost:8080/ui/](http://localhost:8080/ui/).

### Kubernetes

If you are planning on deploying the server in a Kubernetes cluster, please check the [doc/kubernetes](doc/kubernetes/README.md) documentation for manifests examples and more documentation.

## Configuration

All configurations are done via environment variables.

### General

- `DEBUG` (optional, boolean, defaults to `false`): activate debug logs. Beware that when debug logs are activated, **basic auth username/password and database passwords will exposed in logs**.
- `LISTEN_ON` (optional, string, defaults to `:8080`): which IP/Port the server should listen on. Omitting the IP will make the server listen on all interfaces.
- `MAX_FORM_SIZE` (optional, int, defaults to `102400` - 100KiB): maximum size of requests' multipart form-data (used by `/database`, `/echo/form`, `/request` and `/tcp` endpoints), in bytes.
- `STATIC_FOLDER` (optional, string): the folder path to serve static content. If set, the static content will be accessible under the `/static/` endpoint. This folder must be readable by the server (the docker container is running with user `nobody:nobody`, `65534:65534`).
- `TEMP_FOLDER` (optional, string, defaults to `/tmp/integration-toolbox-webserver`): the folder used by the server to save temporary data, it must be writable by the server. If the folder does not exist, the server attempt to create it at startup.

### Basic Auth

The server offers the ability to protect "all endpoints" (excluding monitoring ones) with basic authentication. To activate it, you must set both of the following environment variables:

- `BASIC_AUTH_USERNAME` (optional, string): username for the basic authentication
- `BASIC_AUTH_PASSWORD` (optional, string): password for the basic authentication.

If one of the two variables is set without the other, the server will return an error and stop.

Once basic authentication is configured, you'll need to add `--user name:password --basic` options to your curl commands.

**curl examples:**

```bash
curl --user name:password --basic http://localhost:8080/download
curl -u name:password --basic http://localhost:8080/download
```

### TLS

Configures the server to listen to HTTPS requests. If set, **all endpoints** will be running under TLS (including monitoring ones).

- `SERVER_TLS_FILE` (optional, string): path to the PEM encoded TLS certificate file.
- `SERVER_TLS_KEY` (optional, string): path to the PEM encoded TLS certificate key file.

If one of the two variable is set without the other, the server will return an error and stop.

### Monitoring

The server exposes 3 monitoring endpoints: `/started`, `/alive` and `/ready` to match Kubernetes' monitoring mechanism (more information in the [official documentation](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#probe-outcome)). Each endpoinds behavior is configurable, either by setting environment variables or by sending a request to the [endpoint](#post-started). In each following environment variables, the prefix `STARTUP` can be replaced by `LIVENESS` or `READINESS` to configure respective endpoints.

- `STARTUP_PROBE_STATUS_OK` (optional, int, defaults to `200`): which HTTP status code should be returned when the check is successful.
- `STARTUP_PROBE_STATUS_ERROR` (optional, int, defaults to `500`): which HTTP status code should be returned when the check fails.
- `STARTUP_PROBE_FAIL` (optional, boolean, defaults to `false`): tells if the endpoint should fail check requests.
- `STARTUP_PROBE_FAIL_NB` (optional, int, defaults to `0`): number of times the endpoint should fail check requests before returning a success. If this is configured while `STARTUP_PROBE_FAIL` is set to `true`, this number will only start decreasing after the endpoint is reconfigured to pass checks.
- `STARTUP_PROBE_DELAY` (optional, [Golang duration](https://pkg.go.dev/time#ParseDuration), defaults to `0`): the duration the endpoint should wait before answering to check requests (both failed and success ones).

## Endpoints

When not specified the HTTP method is not checked by the endpoint, meaning that the endpoint will be accessible whatever the HTTP method used.


### `/crash`

Asks the server to stop, with an optional exit code.

**Query parameters:**

- `code` (optional, int, default to `1`): the exit code the server should crash with.
- `timeout` (optional, [Golang duration](https://pkg.go.dev/time#ParseDuration), defaults to `1s`): the timeout before the server crashes.

**Returned status codes:**

- `HTTP/Ok 200`: ok, the server will crash.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl examples:**

```bash
curl http://localhost:8080/crash # the server will stop with exit code 1
curl http://localhost:8080/crash?code=2 # the server will stop with exit code 2
curl "http://localhost:8080/crash?code=3&timeout=1m" # the server will stop in 1 minute with exit code 3
```

### `/download`

Asks the server to generate some data to download. The generated data will be in binary form, and will only contain `0x00` bytes.

**Query parameters**

- `size` (optional, int, defaults to `1048576` - 1MiB): the size of the content to download.

**Returned status codes:**

- `HTTP/Ok 200`: ok, the server will send the data to download.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl examples:**

```bash
curl http://localhost:8080/download
curl http://localhost:8080/download?size=5242880 # 5MiB
```

### `/echo`

Asks the server to echo (in the answer body) the content of the request (HTTP headers and request body).

**Query parameters**

- `headers` (optional, boolean, defaults to `false`): tells the server to also echo request headers.

**Returned status codes:**

- `HTTP/Ok 200`: ok, the server will echo the request.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl example:**

```bash
curl -H "Some-Header: HeaderValue" -d "This is the request payload" http://localhost:8080/echo?headers=true
```
will return:
```
--- REQUEST HEADERS
POST /echo HTTP/1.1
Host: localhost:8080
Content-Length: 27
Content-Type: application/x-www-form-urlencoded
User-Agent: curl/7.88.1
Accept: */*
Some-Header: HeaderValue

--- BODY
This is the request payload
```

### `/echo/form`

Asks the server to echo (in the answer body) the content of the posted form (multipart-data). The maximum size of the form data is controlled by the [`MAX_FORM_SIZE`](#general) environment variable.

Event though the HTTP method is not checked, `POST` should be prefered since parameters are sent using a multipart form-data (as per [RFC 1867](https://datatracker.ietf.org/doc/html/rfc1867)).

**Query parameters:**

- `headers` (optional, boolean, defaults to `false`): tells the server to also echo request headers.

**Returned status codes:**

- `HTTP/Ok 200`: ok, the server will echo the request.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl example:**

```bash
curl -F key1=value1 -F key2=value2 http://localhost:8080/echo/form?headers=true
```
will return:
```
--- REQUEST HEADERS
POST /echo/form HTTP/1.1
Host: localhost:8080
User-Agent: curl/7.88.1
Accept: */*
Content-Length: 244
Content-Type: multipart/form-data; boundary=------------------------0a0bcc6aeb20e788

--- FORM
headers: true
key1: value1
key2: value2

```

### `/echo/raw`

Asks the server to echo the request.

**Query parameters:**

- `headers` (optional, boolean, defaults to `false`): tells the server to also echo request headers. If set to true, request headers will be copied in answer headers (which will also contain some server specific ones, such as `Date`).

**Returned status codes:**

- `HTTP/Ok 200`: ok, the server will echo the request.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl example:**

```bash
curl -v -H "Some-Header: HeaderValue" -d "This is the request payload" http://localhost:8080/echo/raw?headers=true # note the "-v" option to see the full HTTP request and answer
```
will return:
```
* Uses proxy env variable no_proxy == 'localhost,127.0.0.0/8,::1'
*   Trying 127.0.0.1:8080...
* Connected to localhost (127.0.0.1) port 8080 (#0)
> POST /echo/raw?headers=true HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.88.1
> Accept: */*
> Some-Header: HeaderValue
> Content-Length: 27
> Content-Type: application/x-www-form-urlencoded
>
< HTTP/1.1 200 OK
< Accept: */*
< Content-Length: 27
< Content-Type: application/x-www-form-urlencoded
< Some-Header: HeaderValue
< User-Agent: curl/7.88.1
< Date: Thu, 23 May 2024 06:39:17 GMT
<
* Connection #0 to host localhost left intact
This is the request payload
```

### `/ping`

Ping an distant server. The result of the ping will be returned in the answer body. The ping timeout is 20 seconds.

**Query parameters:**

- `host` (mandatory, string): DNS or IP of the distant server to ping.
- `count` (optional, int, defaults to `3`): number of ping requests to send.

**Returned status codes:**

- `HTTP/Ok 200`: ping has been performed.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl example:**

```bash
curl "http://localhost:8080/ping?host=google.com&count=10"
```
will return:
```
ping results: sent: 10, received: 10 (100.00%), min timing: 26.762635ms, max timing: 226.866164ms, average timing: 48.59647ms
```

For the ping to work on a Linux OS, please take a look at the [run](#run) section.

### `/request`

Asks the server to perform a request on the network. It's compatible with both basic HTTP and websocket connection (with or without TLS). In the case of a websocket request, the connection will be closed right after being opened. It is also possible to send a request over a proxy.

Event though the HTTP method is not checked, `POST` should be prefered since parameters are sent using a multipart form-data (as per [RFC 1867](https://datatracker.ietf.org/doc/html/rfc1867)).

**Form parameters:**

- `url` (mandatory, string): the URL to reach out to. This URL must contain the scheme (`http://`, `https://`, `ws://` or `wss://`) and the optional port.
- `method` (optional, string, defaults to `GET`): HTTP method to use. This option will be ignored when opening websocket connections since the method must be `GET`, as per [RFC 6455](https://datatracker.ietf.org/doc/html/rfc6455).
- `tls_insecure` (optional, boolean, defaults to `false`): asks the server to not verify remote server's certificate.
- `tls_ca` (optional, string, defaults to system's ones): the CA certificate used to verify remote server's certificate, PEM encoded. The certificate can be sent both as a value or as a file.
- `tls_user_cert` (optional, string): the certificate the server should to authenticate itself against the remote server, PEM encoded. The client certificate can be sent both as a value or as a file. If set, the `tls_user_key` parameter must also be set.
- `tls_user_key` (optional, string): the certificate key the server should to authenticate itself against the remote server, PEM encoded. The client certificate key can be sent both as a value or as a file. If set, the `tls_user` parameter must also be set.
- `proxy_url` (optional, string): the URL of the proxy. The URL must contain the scheme (`http://` or `https://`).
- `proxy_username` (optional, string): username used to authenticate against the proxy.
- `proxy_password` (optional, string): password used to authenticate against the proxy.
- `connection_timeout` (optional, [Golang duration](https://pkg.go.dev/time#ParseDuration), defaults to `20s`): the timeout to establish connection with the remote server.
- `echo_headers` (optional, boolean, defaults to `false`): should the request answer headers be returned in the answer body.
- `echo_body` (optional, boolean, defaults to `false`): should the request answer body be returned in the answer body (most of the time empty when test websockets).

**Returned status codes:**

- `HTTP/Ok 200`: the request ran correctly.
- `HTTP/Bad Request 400`: an error was faced with the configuration or while trying to run the request. The error is returned in the answer body.

**curl example:**

```bash
# simple HTTP request
curl -F url=http://google.com \
  http://localhost:8080/request

# simple HEAD HTTPS request that output both headers and body
curl -F url=https://google.com \
  -F method=HEAD \
  -F echo_headers=true \
  -F echo_body=true \
  http://localhost:8080/request

# simple HTTP request with proxy
curl -F url=http://google.com \
  -F proxy_url=http://myproxy:8080 \
  -F proxy_username=my_user \
  -F proxy_password=my_password \
  http://localhost:8080/request

# simple websocket over TLS
curl -F url=wss://mywebsocketserver.tld/ws \
  http://localhost:8080/request
```

### `/sleep`

Asks the server to delay the response, and to answer with an optionaly defined status code.

**Query parameters:**

- `duration` (optional, [Golang duration](https://pkg.go.dev/time#ParseDuration), defaults to `1s`): the duration the server should wait before returning the answer.
- `code` (optional, int, defaults to `200`): the status code the server should answer with.

**Returned status codes:**

- `HTTP/Ok 200` - or the asked status code: all good.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl examples:**

```bash
curl http://localhost:8080/sleep # the endpoint will sleep for 1 second and will return the 200 status code
curl http://localhost:8080/sleep?duration=15s # the endpoint will sleep for 15 seconds and will return the 200 status code
curl "http://localhost:8080/sleep?duration=10s&code=201" # the endpoint will sleep for 10 seconds and will return the 201 status code
```

### `/tcp`

Asks the server to perform a TCP request on the network.

Event though the HTTP method is not checked, `POST` should be prefered since parameters are sent using a multipart form-data (as per [RFC 1867](https://datatracker.ietf.org/doc/html/rfc1867)).

**Form parameters:**

- `host` (mandatory, string): the host and port to open TCP connection to. The must not contain the scheme (i.e.: `tcp://`).
- `tls_enabled` (optional, boolean, defaults to `false`): asks the server to use TLS for opening the connection.
- `tls_insecure` (optional, boolean, defaults to `false`): asks the server to not verify remote server's certificate.
- `tls_ca` (optional, string, defaults to system's ones): the CA certificate used to verify remote server's certificate, PEM encoded. The certificate can be sent both as a value or as a file.
- `tls_user_cert` (optional, string): the certificate the server should to authenticate itself against the remote server, PEM encoded. The client certificate can be sent both as a value or as a file. If set, the `tls_user_key` parameter must also be set.
- `tls_user_key` (optional, string): the certificate key the server should to authenticate itself against the remote server, PEM encoded. The client certificate key can be sent both as a value or as a file. If set, the `tls_user` parameter must also be set.
- `connection_timeout` (optional, [Golang duration](https://pkg.go.dev/time#ParseDuration), defaults to `20s`): the timeout to establish connection with the remote server.
- `echo_body` (optional, boolean, defaults to `false`): should the data returned by the remote server be returned in the answer body. If the remote server does not send back any data (such as a HTTP servers for instance, since the client should be the first one sending data), the endpoint will hang until the read timeout (same value as `connection_timeout`) is reached. Beware that the output may be in binary format and mess up your terminal.
- `echo_body_size` (optional, int, defaults to `1048576` - 1MiB): the maximum size of the echo body, in bytes.

**Returned status codes:**

- `HTTP/Ok 200`: the TCP opened correctly.
- `HTTP/Bad Request 400`: an error was faced with the configuration or while trying to open the TCP connection. The error is returned in the answer body.

**curl example:**

```bash
# simple TCP request
curl -F host=postgresql:5432 \
  http://localhost:8080/tcp

# simple TCP request over TLS without verifying remote certificate
curl -F url=postgresql:5432 \
  -F tls_enabled=true \
  -F tls_insecure=true \
  http://localhost:8080/request

# simple TCP request which output the data sent from the remote server, up to 102400 bytes (100KiB)
curl -F url=postgresql:5432 \
  -F echo_body=true \
  -F echo_body_size=102400 \
  http://localhost:8080/request
```

### `/status_code`

Asks the server to respond with specific status code.

**Query parameters:**

- `code` (mandatory, int): the status code the server should answer with.

**Returned status codes:**

- `asked status code`: all good.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl example:**

```bash
curl http://localhost:8080/status_code?code=201 # the endpoint will answer with the 201 status code
```

### `POST /database/connect`

Asks the server to try to connect to a database backend. The server supports [MySQL](https://www.mysql.com/), [PostgreSQL](https://www.postgresql.org/) and [Microsoft SQL Server](https://www.microsoft.com/en-us/sql-server) database engines.

Event though the HTTP method is not checked, `POST` should be prefered since parameters are sent using a multipart form-data (as per [RFC 1867](https://datatracker.ietf.org/doc/html/rfc1867)).

**Form parameters:**

- `engine` (mandatory, string): what database driver should be used. Put `mysql` for MySQL, `postgres` for PostgreSQL or `sqlserver` for Microsoft SQL Server.
- `host` (optional, string): host of the database server.
- `port` (optional, int, defaults to database engine default): port on which the database server is listening.
- `username` (optional, string): username used to authenticate against the database server.
- `password` (optional, string): password used to authenticate against the database server. If set, `username` must also be set.
- `db_name` (optional, string): name of the database to connect to.
- `tls_enabled` (optional, boolean, defaults to `false`): asks the server to use TLS for opening the connection.
- `tls_insecure` (optional, boolean, defaults to `false`): asks the server to not verify database's certificate. This option is not available with the `postgres` driver, but a similar behavior is possible by setting the SSL mode to `require`.
- `ssl_mode` - PostgreSQL only - (mandatory when `tls_enabled` is set to `true`, string): which TLS mode to use. Must be one of: `require`, `verify-ca`, `verify-full` (other modes deactivate TLS). For more information, please read the [official documentation](https://www.postgresql.org/docs/current/libpq-ssl.html).
- `tls_ca` (optional, string, defaults to system's ones): the CA certificate used to verify database's certificate, PEM encoded. The certificate can be sent both as a value or as a file. This will not have any effect when using the `require` TLS mode with PostgreSQL.
- `tls_user_cert` (optional, string): the certificate the server should to authenticate itself against the database server, PEM encoded. The client certificate can be sent both as a value or as a file. If set, the `tls_user_key` parameter must also be set. This option is not supported with Microsoft SQL Server.
- `tls_user_key` (optional, string): the certificate key the server should to authenticate itself against the database server, PEM encoded. The client certificate key can be sent both as a value or as a file. If set, the `tls_user` parameter must also be set. This option is not supported with Microsoft SQL Server.

**Returned status codes:**

- `HTTP/Ok 200`: database conection established.
- `HTTP/Bad Request 400`: an error was faced with the configuration or while trying to connect to the database. The error is returned in the answer body.

**curl examples:**

```bash
# simple test, without TLS
curl -F engine=mysql \
    -F host=127.0.0.1 \
    -F username=myuser \
    -F password=myuserpassword \
    -F db_name=mydb \
    http://localhost:8080/database/connect

# test with TLS
curl -F engine=postgres \
    -F host=127.0.0.1 \
    -F username=myuser \
    -F password=myuserpassword \
    -F db_name=mydb \
    -F tls_enabled=true \
    -F ssl_mode=require \
    http://localhost:8080/database/connect

# test with TLS and CA
curl -F engine=postgres \
    -F host=127.0.0.1 \
    -F username=myuser \
    -F password=myuserpassword \
    -F db_name=mydb \
    -F tls_enabled=true \
    -F ssl_mode=verify-full \
    -F tls_ca=@/path/to/dbserver/ca.crt \
    http://localhost:8080/database/connect

# test with TLS, CA and user certs
curl -F engine=postgres \
    -F host=127.0.0.1 \
    -F username=myuser \
    -F password=myuserpassword \
    -F db_name=mydb \
    -F tls_enabled=true \
    -F ssl_mode=verify-full \
    -F tls_ca=@/path/to/dbserver/ca.crt \
    -F tls_user_cert=@/path/to/client/cert.crt \
    -F tls_user_key=@/path/to/client/key.key \
    http://localhost:8080/database/connect
```

### `POST /database/query`

Asks the server to try to run a query on a database backend. The server supports [MySQL](https://www.mysql.com/), [PostgreSQL](https://www.postgresql.org/) and [Microsoft SQL Server](https://www.microsoft.com/en-us/sql-server) database engines.

Event though the HTTP method is not checked, `POST` should be prefered since parameters are sent using a multipart form-data (as per [RFC 1867](https://datatracker.ietf.org/doc/html/rfc1867)).

**Form parameters:**

- `engine` (mandatory, string): what database driver should be used. Put `mysql` for MySQL, `postgres` for PostgreSQL or `sqlserver` for Microsoft SQL Server.
- `host` (optional, string): host of the database server.
- `port` (optional, int, defaults to database engine default): port on which the database server is listening.
- `username` (optional, string): username used to authenticate against the database server.
- `password` (optional, string): password used to authenticate against the database server. If set, `username` must also be set.
- `db_name` (optional, string): name of the database to connect to.
- `tls_enabled` (optional, boolean, defaults to `false`): asks the server to use TLS for opening the connection.
- `tls_insecure` (optional, boolean, defaults to `false`): asks the server to not verify database's certificate. This option is not available with the `postgres` driver, but a similar behavior is possible by setting the SSL mode to `require`.
- `tls_mode` - PostgreSQL only - (mandatory when `tls_enabled` is set to `true`, string): which TLS mode to use. Must be one of: `require`, `verify-ca`, `verify-full` (other modes deactivate TLS). For more information, please read the [official documentation](https://www.postgresql.org/docs/current/libpq-ssl.html).
- `tls_ca` (optional, string, defaults to system's ones): the CA certificate used to verify database's certificate, PEM encoded. The certificate can be sent both as a value or as a file. This will not have any effect when using the `require` TLS mode with PostgreSQL.
- `tls_user_cert` (optional, string): the certificate the server should to authenticate itself against the database server, PEM encoded. The client certificate can be sent both as a value or as a file. If set, the `tls_user_key` parameter must also be set. This option is not supported with Microsoft SQL Server.
- `tls_user_key` (optional, string): the certificate key the server should to authenticate itself against the database server, PEM encoded. The client certificate key can be sent both as a value or as a file. If set, the `tls_user` parameter must also be set. This option is not supported with Microsoft SQL Server.
- `query` (mandatory, string): the query to run on the database server.

**Returned status codes:**

- `HTTP/Ok 200`: the query ran correctly.
- `HTTP/Bad Request 400`: an error was faced with the configuration or while trying to run the query against the database. The error is returned in the answer body.

**curl example:**

```bash
curl -F engine=mysql \
    -F host=127.0.0.1 \
    -F username=myuser \
    -F password=myuserpassword \
    -F db_name=mydb \
    -F query="SELECT true;" \
    http://localhost:8080/database/query # will try to run 'SELECT true;' on the database server
```

Examples with TLS are available in the [`POST /database/connect`](#post-databaseconnect) section.

### `/cpu/load`

Asks the server to create some CPU load. To create some load the CPU starts a worker that runs an empty `while(true){}` loop.

**Query parameters:**

- `nb_threads` (optional, int, defaults to `1`): the number of load workers (threads) the server should start. If set to `0` the server will start as many load workers as the number of CPU cores present on the server.
- `timeout` (optional, [Golang duration](https://pkg.go.dev/time#ParseDuration), defaults to `0`): the duration the server should wait between two iterrations of the loop. If set too high, the server CPU usage increase will not be noticable.

**Returned status codes:**

- `HTTP/Ok 200`: a load worker has been started.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl example:**

```bash
curl http://localhost:8080/cpu/load # will start one load worker - you should see 1 server's CPU core hitting 100% usage
curl http://localhost:8080/cpu/load?nb_threads=0 # will start as many load workers as CPU cores present in the system - you should see all server's CPU cores hitting 100% usage
curl "http://localhost:8080/cpu/load?nb_threads=2&timeout=1ms" # will start 2 load workers with a 1ms timeout between each loop iteration - you should see an increase of CPU usage on 2 of the server CPU cores
```

### `/cpu/reset`

Asks the server to stop all previously started load workers.

**Returned status codes:**

This endpoint will always return the `HTTP/Ok 200` status code.

**curl example:**

```bash
curl http://localhost:8080/cpu/reset # server's CPU usage should be back to normal
```

### `/ram/increase`

Asks the server to increase its RAM usage, and will return the current memory usage in the response body.

**Query parameters:**

- `size` (optional, int, defaults to `1048576` - 1MiB): the amout of memory to allocate, in bytes.

**Returned status codes:**

- `HTTP/Ok 200`: a the memory usage has bee increased.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl example:**

```bash
curl http://localhost:8080/ram/increase # the server will increase memory usage by 1MiB
curl http://localhost:8080/ram/increase?size=5242880 # the server will increase memory usage by 5MiB
```

### `/ram/decrease`

Asks the server to decrease its RAM usage, from previous increase requests, and will return the current memory usage in the response body.

**Query parameters:**

- `size` (optional, int, defaults to `1048576` - 1MiB): the amout of memory to deallocate, in bytes.

**Returned status codes:**

- `HTTP/Ok 200`: a the memory usage has bee decreased.
- `HTTP/Partial Content 206`: a the memory usage has bee decreased but not up to the requested size.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl example:**

```bash
curl http://localhost:8080/ram/decrease # the server will decrease memory usage by 1MiB
curl http://localhost:8080/ram/decrease?size=5242880 # the server will decrease memory usage by 5MiB
```

### `/ram/leak`

Asks the server to simulate a memory leak.

**Query parameters:**

- `size` (optional, int, defaults to `1048576` - 1MiB): the amout of memory to leak per frequency, in bytes.
- `frequency` (optional, [Golang duration](https://pkg.go.dev/time#ParseDuration), defaults to `0`): the frequency at which the server should leak memory.

**Returned status codes:**

- `HTTP/Ok 200`: a memory leak work has been started.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl example:**

```bash
curl http://localhost:8080/ram/leak # the server will leak memory at a rate of 1MiB per round (pretty fast)
curl http://localhost:8080/ram/leak?frequency=1s # the server will leak memory at a rate of 1MiB/s
curl "http://localhost:8080/ram/leak?size=5242880&frequency=0.1s" # the server will leak memory at a rate of 5MiB/0.1s (50MiB/s)
```

### `/ram/reset`

Asks the server to release all previously allocated memory, and stop all started leaks. The server will also return the current memory usage in the response body.

You may need to call this endpoint multiple time to force Golang's garbage collector to release the allocated RAM back to the OS.

**Returned status codes:**

This endpoint will always return the `HTTP/Ok 200` status code.

**curl example:**

```bash
curl http://localhost:8080/ram/reset # server's RAM usage should be back to normal
```

### `/ram/status`

Asks the server to give a status about its memory usage, sent in the answer body.

**Returned status codes:**

This endpoint will always return the `HTTP/Ok 200` status code.

**curl example:**

```bash
curl http://localhost:8080/ram/status # will return: "memory status: Alloc: 463.48 KiB"
```

### `/static/`

Base endpoint to access the configured static folder. This endpoint will not be activated if the `STATIC_FOLDER` environment variable has not been set.

If you have a file named `index.html`, it will also be served at the root of the path (i.e.: `/static/`).

Static content is served using Golang's [FileServer](https://pkg.go.dev/net/http#FileServer).

### `/ui/`

Home path to the web interface. From there you will be able to access most of the server endpoints, as well as a tool to test CORS resources.

### `/started`

#### GET `/started`

Checks the monitoring endpoint status.

**Returned status codes:**

- `HTTP/Ok 200` - or configured ok status: check succeeded.
- `HTTP/Internal Server Error 500`- or configured failed status: check failed.

**curl example:**

```bash
curl http://localhost:8080/started # will return 200 if fail is set to 'false' and the number of failures is 0, 500 otherwise (if codes have not been changed in the configuration)
```

#### POST `/started`

Modifies the monitoring endpoint behavior.

**Query parameters**

- `fail` (optional, boolean): tells if the endpoint should fail check requests.
- `nb_failures` (optional, int): the number of times the endpoint should fail checks before answering successfully. If `fail` has been set to `true`, this number will start decreasing only after `fail` is set back to `false`. Setting it to 0, will reset the counter.
- `delay` (optional, [Golang duration](https://pkg.go.dev/time#ParseDuration)): the duration the endpoint should wait before answering to check requests.

**Returned status codes:**

- `HTTP/Ok 200`: endpoint has been configured.
- `HTTP/Bad Request 400`: failed to parse one of the query parameter. The error is returned in the answer body.

**curl examples:**

```bash
curl -XPOST http://localhost:8080/started?fail=true # future checks will fail
curl -XPOST http://localhost:8080/started?nb_failures=0 # resets the number of failed requests
curl -XPOST "http://localhost:8080/started?fail=false&nb_failures=3" # future 3 checks will fails, then checs will be ok
curl -XPOST "http://localhost:8080/started?fail=true&delay=10s" # future checks will take 10 seconds to fail
```

### `/alive`

Works the same way as the [/started](#started) endpoint.

### `/ready`

Works the same way as the [/started](#started) endpoint.

## Contribution

Contributions are always welcome. If you want to take part in improving this project, please:

* Fork the repo
* Create a pull request against master

## License

The Integration Toolbox WebServer is released under the [GPL3 license](LICENSE), allowing you to use and modify it freely for your testing needs.

## Credits

### Authors

- Kanshiroron [![Follow on X](https://img.shields.io/twitter/follow/AntoineKanshi)](https://x.com/AntoineKanshi)

### Dependencies

- [github.com/go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) the MySQL driver library
- [github.com/gorilla/websocket](https://github.com/gorilla/websocket) websocket library
- [github.com/lib/pq](https://github.com/lib/pq) the PostgreSQL library
- [github.com/microsoft/go-mssqldb](https://github.com/microsoft/go-mssqldb) the Microsoft SQL Server driver library
- [github.com/pkg/errors](https://github.com/pkg/errors) error wrapping library
- [github.com/prometheus-community/pro-bing](https://github.com/prometheus-community/pro-bing) the ping library
- [github.com/sirupsen/logrus](https://github.com/sirupsen/logrus) logger

### Other

- Logo created by [Freepik](https://www.flaticon.com/free-icon/software-testing_10435234).
