FROM golang:alpine as go-builder

RUN apk add --no-cache make gcc musl-dev linux-headers

RUN addgroup -S app \
    && adduser -S -G app app

WORKDIR /go/src/strategyexecutor


# Copy go.mod and go.sum vendor
COPY --chown=app:app main.go go.mod go.sum ./
COPY --chown=app:app src /go/src/strategyexecutor/src


RUN go build -o strategyexecutor main.go


FROM alpine:3.16
RUN apk add --no-cache tzdata

RUN addgroup -S app \
    && adduser -S -G app app

COPY --from=go-builder --chown=app:app /go/src/strategyexecutor/strategyexecutor /go/src/strategyexecutor/strategyexecutor

RUN chown app:app /go/src/strategyexecutor
USER app
ENTRYPOINT ["/go/src/strategyexecutor/strategyexecutor"]
