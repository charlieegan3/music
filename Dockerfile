FROM golang:1.14 as build

ENV GOPROXY=https://proxy.golang.org

WORKDIR /go/src/github.com/charlieegan3/music

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -o=music

FROM gcr.io/distroless/static:1c3096e506c4cb951f2651a2804fd18fbb84046f

# needed to talk to shazam
COPY geo.crt /etc/ssl/certs/
COPY schema.json /
COPY --from=build /go/src/github.com/charlieegan3/music/music /

ENTRYPOINT ["/music"]
