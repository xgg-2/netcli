# netcli

A terminal-based HTTP/HTTPS traffic inspector — the CLI equivalent of Chrome DevTools' Network tab. `netcli` runs a local MITM proxy that captures traffic from any application or browser and presents it in a real-time interactive TUI.

> **Legal notice:** This tool is intended for inspecting traffic on applications and systems you own or have explicit permission to test. Do not use it to intercept traffic without authorization.

---

## Features

- Live request list with method, status code, host+path, and response time
- Horizontal split TUI: request list on the left, full detail panel on the right
- Color-coded methods (GET/POST/PUT/DELETE) and status codes (2xx/3xx/4xx/5xx)
- Full request and response headers and bodies in the detail panel
- JSON bodies are pretty-printed automatically
- Live filter by host or path substring (`/` key)
- Export any request as a `curl` command to clipboard or stdout
- Save sessions to JSONL files for offline analysis
- Single binary, no runtime dependencies

---

## Requirements

- Go 1.21 or newer (`go version` to check) — only needed if building from source
- Linux, macOS, or Windows

---

## Installation

### Download prebuilt binary

Download the latest release for your platform from the
[Releases page](https://github.com/xgg-2/netcli/releases).

Available binaries:

| File | Platform |
|---|---|
| `netcli-windows-amd64.exe` | Windows (Intel/AMD 64-bit) |
| `netcli-macos-amd64` | macOS, Intel-based Mac |
| `netcli-macos-arm64` | macOS, Apple Silicon (M1/M2/M3/M4) |
| `netcli-linux-amd64` | Linux (Intel/AMD 64-bit) |

**Not sure which macOS binary you need?** Open Terminal and run:
```bash
uname -m
```
`arm64` → download `netcli-macos-arm64`. `x86_64` → download `netcli-macos-amd64`.

**Linux/macOS:**
```bash
chmod +x netcli-linux-amd64
sudo mv netcli-linux-amd64 /usr/local/bin/netcli
```

**Windows:** download `netcli-windows-amd64.exe`, optionally rename it to
`netcli.exe`, and run it from PowerShell or CMD:
```powershell
.\netcli.exe --help
```

> **Note (Windows):** SmartScreen may show a warning ("Windows protected
> your PC") since this binary isn't code-signed with a paid certificate.
> This is expected for open-source binaries distributed this way. Click
> **More info** → **Run anyway** to proceed.

### From source

```bash
git clone https://github.com/xgg-2/netcli
cd netcli
go build -o netcli .
```

Move the binary somewhere on your `PATH`:

```bash
sudo mv netcli /usr/local/bin/
```

### With `go install`

```bash
go install github.com/xgg-2/netcli@latest
```

---

## First-time setup

Generate the local CA certificate (run this once):

```bash
netcli setup
```

This creates a certificate and private key in `~/.config/netcli/`. The command prints OS-specific instructions for trusting the certificate. Trust it once; HTTPS interception then works transparently.

### Trusting the certificate

**macOS:**
```bash
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain ~/.config/netcli/ca.crt
```

**Linux (Debian/Ubuntu):**
```bash
sudo cp ~/.config/netcli/ca.crt /usr/local/share/ca-certificates/netcli.crt
sudo update-ca-certificates
```

**Linux (Fedora/RHEL):**
```bash
sudo cp ~/.config/netcli/ca.crt /etc/pki/ca-trust/source/anchors/netcli.crt
sudo update-ca-trust extract
```

**Windows** (elevated PowerShell — must be run as Administrator):
```powershell
Import-Certificate -FilePath "$env:USERPROFILE\.config\netcli\ca.crt" `
  -CertStoreLocation Cert:\LocalMachine\Root
```

**Firefox** (all platforms): Go to Preferences → Privacy & Security → Certificates → View Certificates → Import.

---

## Usage

### Watch all traffic

```bash
netcli watch
```

Configure your browser or application to use `http://localhost:8080` as its HTTP and HTTPS proxy. All traffic appears in the TUI live.

Options:
```
--port int      proxy listen port (default 8080)
--bind string   IP address to bind the proxy listener to (default "127.0.0.1")
--filter string show only requests matching this domain or path substring
--save string   append captured traffic to a JSONL file
```

By default the proxy binds to `127.0.0.1` (loopback only). To proxy traffic
from another device on the same network, pass `--bind 0.0.0.0`:

```bash
netcli watch --bind 0.0.0.0 --port 8080
```

> **Note:** Exposing the proxy on `0.0.0.0` allows any machine on the local
> network to route traffic through it. Only do this on a trusted network.

Examples:
```bash
# Filter to a specific domain
netcli watch --filter api.example.com

# Save session to file
netcli watch --save session.jsonl

# Custom port + filter + save
netcli watch --port 9090 --filter stripe.com --save stripe-session.jsonl

# Expose proxy to other devices on the local network
netcli watch --bind 0.0.0.0 --port 8080
```

### Scope traffic to a single process

Run `netcli watch` in one terminal, then in another:

```bash
netcli run -- curl https://api.example.com/v1/users
netcli run -- python script.py
netcli run -- node index.js
```

`netcli run` sets `HTTP_PROXY` and `HTTPS_PROXY` only for the child process — no system-wide proxy changes.

### Print certificate info

```bash
netcli cert-info
```

---

## TUI keybindings

| Key           | Action                                              |
|---------------|-----------------------------------------------------|
| `↑` / `↓`    | Navigate the request list                           |
| `PgUp/PgDn`  | Scroll by page                                      |
| `/`           | Open live filter (filters all requests by host/path)|
| `Esc`         | Close filter or save input                          |
| `s`           | Save session (prompts for filename if no `--save`)  |
| `e`           | Export selected request as curl command              |
| `q`           | Quit                                                 |

The detail panel scrolls with the mouse wheel when focused.

---

## Session file format

Each line in a `.jsonl` file is a self-contained JSON record:

```json
{
  "timestamp": "2024-01-15T12:34:56Z",
  "method": "GET",
  "url": "https://api.example.com/v1/users",
  "request_headers": { "Authorization": ["Bearer tok_…"] },
  "request_body": "",
  "status_code": 200,
  "response_headers": { "Content-Type": ["application/json"] },
  "response_body": "{\"users\": []}",
  "duration_ms": 142
}
```

Binary bodies are base64-encoded and prefixed with `base64:`.

> Session files may contain authorization tokens, cookies, and other
> sensitive data. They are written with `0600` permissions (owner-only),
> but treat them as secrets — do not commit them to git or share them
> without redacting sensitive fields first.

---

## Project structure

```
netcli/
├── main.go                  entry point
├── go.mod                   module definition and pinned dependencies
├── go.sum                   dependency checksums
├── cmd/
│   ├── root.go              cobra root command
│   ├── setup.go             netcli setup
│   ├── watch.go             netcli watch
│   ├── run.go                netcli run
│   └── certinfo.go          netcli cert-info
└── internal/
    ├── types/types.go        shared RequestEntry struct
    ├── cert/cert.go          CA generation and path management
    ├── proxy/proxy.go        go-mitmproxy addon and Start()
    ├── export/jsonl.go       JSONL file writer
    └── tui/
        ├── model.go          bubbletea model and Init
        ├── update.go         Update (keyboard handling, state transitions)
        ├── view.go           View (rendering, layout, curl export)
        └── styles.go         lipgloss color and style definitions
```

---

## Dependencies

| Package | Purpose |
|---|---|
| `charmbracelet/bubbletea` | TUI framework |
| `charmbracelet/lipgloss` | Terminal styling |
| `charmbracelet/bubbles` | Reusable TUI components (viewport, textinput) |
| `lqqyt2423/go-mitmproxy` | MITM proxy with TLS interception |
| `spf13/cobra` | CLI command structure |
| `atotto/clipboard` | Clipboard access for curl export |

---

## Certificate pinning

Some applications embed a specific certificate or public key and refuse connections
that present a different certificate, regardless of the system CA store.

Common examples: many mobile apps, some desktop clients (Signal, Spotify), and
Electron apps that ship their own CA bundle.

There is no workaround for certificate pinning without modifying the application
binary. This is intentional security behavior and `netcli` cannot bypass it.

---

## Troubleshooting

**HTTPS traffic not decrypted / shows as CONNECT tunnel**
The CA certificate is not trusted by the target application. Re-run `netcli setup` and follow the trust steps for your OS and browser.

**No requests appearing in the TUI**
Confirm the application is configured to use the proxy. `netcli run` only sets proxy variables for its child process. For browsers, set the proxy in network settings manually.

**Firefox shows a security error**
Firefox uses its own certificate store. Trust the CA separately via Preferences → Privacy & Security → Certificates → Import.

**Address already in use**
Another process is using port 8080. Use `--port` to pick a different port.

---

## Environment variables

`netcli` itself requires no environment variables to operate. The `netcli run` subcommand sets the following variables on the child process — they are never read from the environment:

- `HTTP_PROXY`
- `HTTPS_PROXY`
- `http_proxy`
- `https_proxy`

No `.env` file is needed.

---

## Known Limitations

- **CA certificate lifespan:** The generated CA certificate is valid for 10 years. There is no automatic rotation mechanism. If the key is compromised, manual deletion of `~/.config/netcli/` and re-running `netcli setup` is required.
- **CA private key stored unencrypted:** The key at `~/.config/netcli/ca.key` is protected only by file-system permissions (`0600`). It is not passphrase-protected. Any process running as the same user can read it directly.
- **Unbounded in-memory accumulation:** All captured requests and response bodies are held in memory for the lifetime of the session. Very large payloads or very long watch sessions will increase memory usage continuously. There is currently no eviction or streaming to disk.
- **go-mitmproxy pinned to v1.8.11:** The dependency is pinned because v1.9.x requires Go 1.26, which is not yet the project's minimum toolchain. It will be upgraded once the project moves to Go 1.26+.

---

## Contributing

Contributions are welcome. Please open an issue before submitting a large pull request so the direction can be discussed. Keep changes focused and match the existing code style (no inline comments, no emoji in code or output).

---

## License

MIT — see [LICENSE](LICENSE).
