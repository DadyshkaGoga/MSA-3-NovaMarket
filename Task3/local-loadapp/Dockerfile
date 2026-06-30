# Local arm64 build of the scaletestapp stand-in (the official image is amd64-only).
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY main.go .
RUN go mod init loadapp && CGO_ENABLED=0 go build -o /loadapp .

FROM alpine:3.20
COPY --from=build /loadapp /loadapp
EXPOSE 8080
ENTRYPOINT ["/loadapp"]
