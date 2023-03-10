FROM golang:1.20.2-alpine3.17 as build

WORKDIR /go/src/jollz-twitch-rank

COPY go.mod ./
COPY go.sum ./
COPY libs ./libs

RUN go mod download

WORKDIR /go/src/jollz-twitch-rank/services/twitchbot

COPY services/twitchbot/* ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o twitchbot .

FROM alpine:latest
WORKDIR /root/
COPY --from=build /go/src/jollz-twitch-rank/services/twitchbot/twitchbot ./
COPY --from=build /go/src/jollz-twitch-rank/services/twitchbot/config.json ./
CMD ["./twitchbot"]