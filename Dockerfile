# build the binary
FROM golang:1.13.8 AS builder
# bring in all the packages
COPY main.go /go/src/github.com/amazeeio/lagoon/services/backuphandler/
COPY go.mod /go/src/github.com/amazeeio/lagoon/services/backuphandler/
COPY go.sum /go/src/github.com/amazeeio/lagoon/services/backuphandler/
COPY internal /go/src/github.com/amazeeio/lagoon/services/backuphandler/internal/
WORKDIR /go/src/github.com/amazeeio/lagoon/services/backuphandler/

# tests currently don't work because mocking rabbit is interesting
# RUN GO111MODULE=on go test ./...
# compile
RUN CGO_ENABLED=0  GOOS=linux GOARCH=amd64 go build -a -o backuphandler .

# put the binary into container
# use the commons image
ARG IMAGE_REPO
FROM ${IMAGE_REPO:-lagoon}/commons

WORKDIR /app/

# bring the auto-idler binary from the builder
COPY --from=builder /go/src/github.com/amazeeio/lagoon/services/backuphandler/backuphandler .

ENV LAGOON=backuphandler
# set defaults
ENV JWT_SECRET=super-secret-string \
    JWT_AUDIENCE=api.dev \
    GRAPHQL_ENDPOINT="http://api:3000/graphql"

ENTRYPOINT ["/sbin/tini", "--", "/lagoon/entrypoints.sh"]
CMD ["/app/backuphandler"]