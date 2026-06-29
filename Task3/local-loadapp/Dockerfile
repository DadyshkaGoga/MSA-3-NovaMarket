# Self-contained build of the local arm64 stand-in for scaletestapp (live run only).
# The production deployment.yaml references the official image; this one is a native
# replacement for Apple Silicon, where the amd64 image does not run.
#
#   docker build -t scaletestapp-local:arm64 .
#   minikube image load scaletestapp-local:arm64
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY main.go .
RUN go mod init loadapp && CGO_ENABLED=0 go build -o /loadapp .

FROM alpine:3.20
COPY --from=build /loadapp /loadapp
EXPOSE 8080
ENTRYPOINT ["/loadapp"]
