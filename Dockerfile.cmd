FROM golang:alpine as go-builder

RUN apk add --no-cache make gcc musl-dev linux-headers

RUN addgroup -S app \
    && adduser -S -G app app

WORKDIR /go/src/strategyexecutorcmd


# Copy go.mod and go.sum vendor
COPY --chown=app:app go.mod go.sum ./
COPY --chown=app:app src /go/src/strategyexecutorcmd/src

COPY --chown=app:app cmd /go/src/strategyexecutorcmd/cmd

RUN GOBIN=/go/src/strategyexecutorcmd/bin go install ./cmd/...


FROM alpine:3.16

RUN addgroup -S app \
    && adduser -S -G app app

COPY --from=go-builder --chown=app:app /go/src/strategyexecutorcmd/bin/cmd /go/src/strategyexecutorcmd/bin/cmd

RUN chown app:app /go/src/strategyexecutorcmd/bin/cmd
USER app
ENTRYPOINT ["/go/src/strategyexecutorcmd/bin/cmd"]
