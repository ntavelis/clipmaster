# Passphrase-Based Discovery + HTTPS

## Context
Currently, all Clipmaster instances on the LAN discover each other via mDNS and share clipboard data over plain HTTP with no authentication. This means any Clipmaster on the network can read any other's clipboard. We need:
1. A **passphrase** that gates which peers can see and talk to each other
2. **TLS encryption** on the sync HTTP server so clipboard data is encrypted in transit

## Config File: `~/.config/clipmaster/config.json`

### New package: `foundation/config/config.go`
- Define a `Config` struct with a `Passphrase string` field
- `Load(path string) (Config, error)` — reads and unmarshals `config.json`
- `Save(path string, cfg Config) error` — marshals and writes `config.json` (creates parent dirs if needed)

### Config path in `main.go` appConfig
- Add `ConfigPath string` with `conf:"default:~/.config/clipmaster/config.json"` to `appConfig`
- Overridable via `CLIPMASTER_CONFIG_PATH` env var
- Resolve `~` to `$HOME` at runtime before passing to `foundation/config`
- Pass this path through `app.Config` to `app/app.go`

### First-run prompt in `app/app.go` Startup
- Before starting sync/discovery, check if config file exists
- If not, use `runtime.InputDialog` (Wails native dialog) to ask for the passphrase
- Save it to `config.json`
- If config exists, load it
- Store passphrase in the `App` struct and pass it down to mDNS, sync server, and peer fetcher

## Passphrase Filtering — Two Layers

### Layer 1: mDNS TXT record hash
**File:** `foundation/mdns/mdns.go`
- Accept a `passphrase` parameter in `New()` and `Register()`
- Compute `SHA-256(passphrase)`, take first 16 hex chars as `passphraseHash`
- Include `ph=<hash>` in the mDNS TXT record (alongside existing `version=1`)
- In `browse()`, when iterating discovered entries, parse TXT records and **skip** any peer whose `ph` value doesn't match ours
- This prevents peers with different passphrases from even appearing in the peer list

### Layer 2: HTTP header validation
**File:** `app/handlers/clipboard.go`
- Check for an `X-Clipmaster-Pass` header on incoming requests
- Compare its value against the local passphrase (constant-time comparison via `crypto/subtle`)
- Return `401 Unauthorized` if missing or mismatched

**File:** `business/peersclipsync/peersclipsync.go`
- Accept passphrase in `New()`
- Include `X-Clipmaster-Pass: <passphrase>` header on every `GET /api/clipboard` request

## HTTPS with Self-Signed TLS

### New package: `foundation/tlscert/tlscert.go`
- `Generate() (tls.Certificate, error)` — generates a self-signed X.509 cert + RSA key in memory (no files written to disk)
  - Uses `crypto/x509`, `crypto/rsa`, `crypto/rand`
  - Cert valid for 1 year, SAN includes all local IPs + `localhost`
  - Returns a `tls.Certificate` ready for use

### Server-side: `business/sync/server.go`
- Accept a `tls.Certificate` in `New()` or `Start()`
- Change `Start()` to use `tls.NewListener(ln, &tls.Config{Certificates: []tls.Certificate{cert}})` wrapping the raw TCP listener
- Then call `s.server.Serve(tlsListener)` — this makes the server HTTPS
- No other changes needed; same mux, same handlers

### Client-side: `business/peersclipsync/peersclipsync.go`
- Create the `http.Client` with a custom `Transport` that has `TLSClientConfig: &tls.Config{InsecureSkipVerify: true}`
- Change the URL scheme from `http://` to `https://` in `fetchPeer()`

## Wiring in `app/app.go` and `main.go`

### `main.go`
- Add config file path to `app.Config` (use `foundation/config.DefaultPath()`)

### `app/app.go` Startup flow (updated order)
1. Load or prompt for passphrase (using `foundation/config` + Wails dialog)
2. Generate TLS cert (`foundation/tlscert`)
3. Create sync server with TLS cert
4. Register routes (pass passphrase to handler for validation)
5. Start sync server (now HTTPS)
6. Register mDNS with passphrase hash in TXT
7. Start discoverer (filters by passphrase hash)
8. Start peer fetcher (sends passphrase header, uses skip-verify HTTPS client)
9. Start clipboard monitor (unchanged)

## Files to Create
| File | Purpose |
|------|---------|
| `foundation/config/config.go` | Config load/save for `config.json` |
| `foundation/tlscert/tlscert.go` | In-memory self-signed TLS cert generation |

## Files to Modify
| File | Changes |
|------|---------|
| `foundation/mdns/mdns.go` | Add passphrase param, hash in TXT, filter in browse |
| `business/sync/server.go` | Accept TLS cert, wrap listener with TLS |
| `business/peersclipsync/peersclipsync.go` | HTTPS client, passphrase header |
| `app/handlers/clipboard.go` | Validate `X-Clipmaster-Pass` header |
| `app/app.go` | Wire config, TLS, passphrase through startup |
| `app/routes.go` | Pass passphrase to handler |
| `main.go` | Add `ConfigPath` to `appConfig` with default, pass to `app.Config` |

## Verification
1. Build and run: `wails dev`
2. First run: native dialog should appear asking for passphrase — enter one and confirm `~/.config/clipmaster/config.json` is created with it
3. Second run: no dialog, passphrase loaded from file
4. Run two instances (different machines or ports) with **same** passphrase — they should discover each other and sync clipboards over HTTPS
5. Run an instance with a **different** passphrase — it should NOT appear in the peer list, and direct HTTPS requests without the correct header should return 401
6. Verify with `curl -k https://localhost:<port>/api/clipboard` returns 401 (no passphrase header)
7. Verify with `curl -k -H "X-Clipmaster-Pass: <correct>" https://localhost:<port>/api/clipboard` returns clipboard JSON
