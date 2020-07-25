FROM golang:1.14 as build

ENV GOPROXY=https://proxy.golang.org

WORKDIR /go/src/github.com/charlieegan3/music

COPY . .

RUN go build -o=music


FROM gcr.io/distroless/base-debian10

COPY schema.json /
COPY --from=build /go/src/github.com/charlieegan3/music/music /

ENTRYPOINT ["/music"]
