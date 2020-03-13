# Wrestic snapshot webhook handler

Build, then deploy. Requires the following vars to start

```
BROKER_ADDRESS=localhost
BROKER_PORT=5672
BROKER_USER=guest
BROKER_PASS=guest
JWT_SECRET="super-secret-string"
JWT_AUDIENCE="api.dev"
GRAPHQL_ENDPOINT="http://localhost:3000/graphql"
```

## Build

```
./build-push ${REPO:-amazeeiolagoon} ${TAG:-latest}
```