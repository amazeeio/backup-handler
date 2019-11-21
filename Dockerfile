# build the binary
FROM golang AS builder
# bring in all the packages
COPY main.go /go/src/github.com/amazeeio/lagoon/services/backuphandler/
COPY go.mod /go/src/github.com/amazeeio/lagoon/services/backuphandler/
COPY go.sum /go/src/github.com/amazeeio/lagoon/services/backuphandler/
WORKDIR /go/src/github.com/amazeeio/lagoon/services/backuphandler/

# get any imports as required
RUN set -x && go get -v .
# compile
RUN CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=amd64 go build -a -o backuphandler .

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