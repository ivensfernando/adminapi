# Biidin Admin API

Biidin Admin API is an HTTP API for logging trades, managing exchange connections, and processing webhook alerts. The service secures client-facing endpoints with a shared `X-Secret-Key` header plus JWT cookies, and exposes a Swagger definition for the latest API contract.

## Features
- Health and readiness check at `/health`.
- JWT-based authentication (register, login, logout, profile update) under `/api/v1/auth` and `/api/v1/me`.
- CRUD APIs for trades at `/api/v1/trades` with bulk delete support.
- Lookup endpoints for supported exchanges and trading pairs under `/api/v1/lookup`.
- User exchange management with credential encryption and connection testing at `/api/v1/user-exchanges`.
- Webhook management for outbound alerts and retrieval of received webhook alerts under `/api/v1/webhooks` and `/api/v1/webhook-alerts`.
- Public trading webhook receiver at `/trading/webhook/{token}` for ingesting alerts without the shared secret header.
- Local Swagger UI served from `docs/swagger.yaml` via `make swagger`.

## Project structure
```
trading-journal/
├── main.go                # HTTP server entrypoint
├── docs/swagger.yaml      # OpenAPI specification for the service
├── src/
│   ├── auth/              # JWT and middleware helpers
│   ├── db/                # Database connection and migrations
│   ├── handler/           # HTTP handlers for auth, trades, webhooks, etc.
│   ├── lookup/            # Lookup providers for exchanges and pairs
│   ├── model/             # GORM models
│   ├── repository/        # Data access layer
│   ├── security/          # Encryption helpers for stored secrets
│   └── server/            # Router and middleware wiring
├── db/migrations/         # SQL migrations (used by migrate CLI)
├── Makefile               # Developer tooling (run, build, swagger, tests)
└── docs/                  # API documentation assets
```

## Prerequisites
- Go 1.23+
- Postgres with connection details provided via environment variables
  - `PGHOST`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `PGPORT`
- Secrets for authentication and request protection
  - `JWT_SECRET` for signing tokens
  - `SHARED_SECRET` for the `X-Secret-Key` middleware

## Installation
1. Clone this repository:
   ```bash
   git clone https://github.com/your-repo/trading-journal.git
   cd trading-journal
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```

## Running the API server
1. Ensure Postgres is reachable and migrations are applied (see [Run the migration](#run-the-migration)).
2. Start the server with the required environment variables:
   ```bash
   APP_NAME=trading-journal PORT=3010 \
   PGHOST=localhost PGUSER=postgres PGPASSWORD=postgres PGDATABASE=trading_journal PGPORT=5432 \
   JWT_SECRET=changeme SHARED_SECRET=supersecret \
   go run main.go
   ```
3. The API will listen on `http://localhost:3010`.

Alternatively, use the Makefile shortcut:
```bash
make run_server
```

## API base path and documentation
- Public health checks remain at `/health`, and trading alerts are accepted at `/trading/webhook/{token}`.
- All other client-facing endpoints (authentication, trades, lookup, user exchanges, webhooks, webhook alerts, etc.) are versioned under `/api/v1`.

To explore the OpenAPI (Swagger) definition in a browser, run a local Swagger UI container that serves the bundled `docs/swagger.yaml`:
```bash
docker run --rm -p 8080:8080 -e SWAGGER_JSON=/swagger.yaml \
  -v "$(pwd)/docs/swagger.yaml:/swagger.yaml" swaggerapi/swagger-ui
```

Or use the Makefile shortcut:
```bash
make swagger
```

Then open http://localhost:8080 in your browser. The UI will load the specification from the mounted `docs/swagger.yaml` file.

## Testing
Run the full test suite:
```bash
go test ./...
```

## KuCoin example (ccxt)
The repository bundles a minimal ccxt-based KuCoin connector under `internal/connectors/kucoin.go`
and a runnable example at `cmd/kucoin/main.go`. Provide your KuCoin API credentials through the
following environment variables:

```
export KUCOIN_API_KEY="<key>"
export KUCOIN_API_SECRET="<secret>"
export KUCOIN_API_PASSPHRASE="<passphrase>"
```

Then run the example to fetch spot and futures balances:

```
make kucoin
# or
go run ./cmd/kucoin/main.go
```

The output prefixes balances with `spot_` or `futures_` to indicate the account type. API access
must be enabled for both spot and futures on your key for the request to succeed.

## Run the migration
```bash
export DATABASE_URL="postgres://user:password@localhost:5432/yourdb?sslmode=disable"
migrate -database "$DATABASE_URL" -path db/migrations up
# to check version
migrate -database "$DATABASE_URL" -path db/migrations version
# to rollback one step (if needed)
# migrate -database "$DATABASE_URL" -path db/migrations down 1
```

If you use Docker Compose, you can run the CLI inside a Postgres or app container; or call migrations from a Makefile/CI step.
## License
This project is licensed under the MIT License.
````     $       
             $
            /!\       
           / ! \     
          /  |  \      
         /   |   \                 