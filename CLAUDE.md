# LabVIEW-AI-Engine-Bridge

Offline-first MCP server in Go. Lets an AI application build LabVIEW VIs from
scratch, and exposes the same tools on an ephemeral local web dashboard
reachable from localhost or a phone on the same Wi-Fi.

```
AI client <--MCP/stdio--> bridge.exe <--TCP 6060--> lv_bridge.vi <--scripting--> LabVIEW
```

## Ground rules
- **Local-first, always.** No cloud calls, no telemetry, no CDN assets, no
  external fonts. The dashboard HTML is embedded in the binary.
- **Ephemeral sessions.** The browser UI keeps state in memory only: no
  localStorage, sessionStorage, cookies, or history writes. Closing the tab
  destroys the session; the server holds nothing per-client but a goroutine.
- **Keep the LabVIEW side small.** Every bit of logic that can live in Go must
  live in Go. Go allocates object ids, computes grid positions, and validates
  arguments, so the bridge VI only parses and dispatches. Never push work onto
  the diagram that Go can do.
- **One creation command.** Controls, indicators, functions and structures all
  go over `CMD:NEW` with a different `CLS`/`STY`. Do not add a second creation
  verb; it costs a Case structure in LabVIEW.
- **Sanitize at the boundary.** `;` and newlines are stripped from every tool
  argument and control types are allowlisted. Colons survive on purpose, because
  Windows paths need them and the daemon splits on the first colon only. Command
  forgery on the daemon socket is the one real attack surface.

## Layout
| File | Role |
|---|---|
| `main.go` | Entry point, stdio transport, MCP routing, tool registry, payload builder, auto-layout |
| `gateway.go` | HTTP server on :8080, `/` dashboard, `/ws` WebSocket pipeline |
| `labview.go` | Persistent TCP client to the bridge VI, plus the `-mock` daemon |
| `discovery.go` | Wi-Fi adapter IP sniff, ASCII/SVG QR code, browser launch |
| `bridge_test.go` | Payload translation, field parsing, RPC dispatch |
| `BRIDGE_VI.md` | How to build the LabVIEW-side addon |
| `README.md` | Install and usage |

## Status
Go side complete. `lv_bridge.vi` is **not** built; see `BRIDGE_VI.md`. Until it
exists, use `-mock` to exercise the chain.

## Commands
```
go mod tidy && go test ./...
go run .                  # dashboard + QR + browser
go run . -mock            # same, with a fake LabVIEW daemon
go run . -stdio           # MCP transport for Claude Desktop / Claude Code
```

## Do not scan / do not read
`*.exe`, `*.dll`, `bin/`, `dist/`, `vendor/`, `node_modules/`, `.git/`, `*.vi`,
`*.lvproj`, `*.llb`, `*.png`, `*.svg`, `*.log`, `tmp/`.

## Conventions
- Stdlib first. Two dependencies only: `gorilla/websocket`, `skip2/go-qrcode`.
- No abstraction with one implementation. No config for a value that never changes.
- Every non-trivial branch keeps a check in `bridge_test.go`.
