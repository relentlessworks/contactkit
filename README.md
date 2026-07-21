# ContactKit

Agentic-first contact management and CRM service. Plain text API, agent-driven, single Go binary with JSON file storage.

## Quick Start

```bash
# Build
make build

# Run (defaults to :7700, stores data in ./contactkit-data.json)
./contactkit

# In dev mode (no SMTP), OTP codes are logged to stderr
# Terminal 1: run the server
./contactkit

# Terminal 2: authenticate
curl -X POST -d "email=me@example.com" http://localhost:7700/auth/request
# Check stderr for OTP code, then:
curl -X POST -d "email=me@example.com&code=123456" http://localhost:7700/auth/verify
# Returns: token=BASE32TOKEN workspace=ws_abc12 email=me@example.com

# Create a contact
curl -X POST -H "Authorization: Bearer TOKEN" \
  -d "name=Jane Doe&email=jane@acme.com&company=Acme&tags=vip,prospect" \
  http://localhost:7700/contacts

# List contacts
curl -H "Authorization: Bearer TOKEN" http://localhost:7700/contacts

# Search contacts
curl -H "Authorization: Bearer TOKEN" "http://localhost:7700/contacts/search?q=acme"

# Get a single contact
curl -H "Authorization: Bearer TOKEN" http://localhost:7700/contacts/contact_xxxxx

# Update a contact
curl -X PUT -H "Authorization: Bearer TOKEN" \
  -d "title=CTO&phone=+1234567890" \
  http://localhost:7700/contacts/contact_xxxxx

# Delete a contact
curl -X DELETE -H "Authorization: Bearer TOKEN" http://localhost:7700/contacts/contact_xxxxx
```

## API Reference

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/request` | Request OTP (body: `email`) |
| POST | `/auth/verify` | Verify OTP, get token (body: `email`, `code`) |
| POST | `/auth/revoke` | Revoke current token (auth required) |

### Contacts

| Method | Path | Description |
|--------|------|-------------|
| POST | `/contacts` | Create contact (body: `name`*, `email`, `phone`, `company`, `title`, `notes`, `tags`) |
| GET | `/contacts` | List all contacts |
| GET | `/contacts/{handle}` | Get single contact |
| PUT | `/contacts/{handle}` | Update contact (partial update) |
| DELETE | `/contacts/{handle}` | Delete contact |
| GET | `/contacts/search?q=` | Search contacts by name, email, company, phone, or tags |

### Other

| Method | Path | Description |
|--------|------|-------------|
| GET | `/help` | API operating manual (also at `/.well-known/agent.md`) |
| GET | `/workspace` | Get workspace info (auth required) |

### Response Format

- **Plain text** (default): One labeled, grepable line per record (e.g. `handle=contact_k7m2q name=Jane Doe email=jane@acme.com`)
- **JSON**: Add `Accept: application/json` header or `?format=json` query param
- **Errors**: `error: message | hint: what to do next`

## Configuration

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `-addr` | `CONTACTKIT_ADDR` | `:7700` | Listen address |
| `-data` | `CONTACTKIT_DATA` | `./contactkit-data.json` | Data file path |
| `-secret` | `CONTACTKIT_SECRET` | (auto-generated) | Token signing secret |
| `-smtp` | `CONTACTKIT_SMTP` | (empty) | SMTP server for OTP emails (empty = log to stderr) |

## Build

```bash
make build    # CGO_ENABLED=0 go build -trimpath
make test     # go test -count=1 -race
make vet      # go vet
make run      # build and run
make clean    # remove build artifacts
```

## Design Principles

- **The agent IS the interface** — No UI, no SDK. The API is the product.
- **Plain text by default** — Token-cheap, grepable, survives context truncation.
- **Instructive errors** — Every 4xx includes a hint for self-correction.
- **Self-documenting** — `GET /help` returns a one-page operating manual.
- **Simple auth** — OTP via email → long-lived bearer token.
- **Single static binary** — Go + JSON file storage, zero external dependencies.
- **Zero config defaults** — Runs out of the box.
- **Multi-tenant ready** — Workspaces isolate data per tenant.
- **Short stable handles** — Every contact addressed by `contact_xxxxx`.

## License

MIT
