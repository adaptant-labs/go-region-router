FROM golang:latest as builder

ENV GO111MODULE=on
ARG REGION_ROUTER_NOTIFICATION_URL

WORKDIR /go/src
ADD . /go/src

RUN go mod download

RUN cd cmd/go-region-router && go build -ldflags "-linkmode external -extldflags -static" -a -o /go/bin/app

FROM scratch
COPY --from=builder /go/bin/app /

EXPOSE 7000
ENTRYPOINT ["/app" ]
