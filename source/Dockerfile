FROM golang:1.22.4-bullseye as base

WORKDIR /

COPY . .

RUN go mod download

RUN go mod verify

RUN GO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /app cmd/main.go

FROM gcr.io/distroless/base-debian12

COPY --from=base /app .

COPY static /static

COPY template /template

CMD ["/app"]
