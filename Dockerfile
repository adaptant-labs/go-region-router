FROM golang:latest as builder

# This is needed in order to avoid an installation outside of $GOPATH error
ENV GOBIN /go/bin

WORKDIR /go/src
ADD . /go/src

RUN go get -v
RUN go build -ldflags "-linkmode external -extldflags -static" -a -o /go/bin/app

FROM scratch
COPY --from=builder /go/bin/app /

EXPOSE 7000
ENTRYPOINT ["/app" ]
