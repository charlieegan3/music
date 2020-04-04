FROM golang:1.14 as build

ENV GOPROXY=https://proxy.golang.org

WORKDIR /go/src/github.com/charlieegan3/music

COPY go.mod go.sum cmd ./

RUN go build -o=musicPlayTracker ./...


FROM scratch
COPY ca-certificates.crt /etc/ssl/certs/
COPY schema.json /
COPY --from=build /go/src/github.com/charlieegan3/music/musicPlayTracker /

CMD ["/musicPlayTracker", "latest"]
