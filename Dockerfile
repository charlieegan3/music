FROM golang:1.10 as build

WORKDIR /go/src/github.com/charlieegan3/music

COPY . .

RUN CGO_ENABLED=0 go build -o musicPlayTracker cmd/*.go


FROM scratch
ADD ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/src/github.com/charlieegan3/music/musicPlayTracker /
COPY --from=build /go/src/github.com/charlieegan3/music/schema.json /

CMD ["/musicPlayTracker", "latest"]
