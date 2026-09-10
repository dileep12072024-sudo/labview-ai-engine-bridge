# LabVIEW-AI-Engine-Bridge

An offline, local-hardware MCP server that lets an AI application build LabVIEW
VIs from scratch. Same shape as Blender MCP: a server the AI talks to, plus an
addon running inside the host application.

```
Claude  <--MCP/stdio-->  bridge.exe  <--TCP 6060-->  lv_bridge.vi  <--scripting-->  LabVIEW
                              |
                              +--HTTP 8080--> phone / desktop dashboard
```

## Install

**1. Nothing to install.** `bridge.exe` is already built for Windows and sits in
this folder at `C:\Users\Dileep\LabVIEW-AI-Engine-Bridge\bridge.exe`. It is a
single static binary with no runtime dependency.

To rebuild it after editing, install Go from https://go.dev/dl (1.22 or newer):

```
go mod tidy
go test ./...
go build -o bridge.exe .
```

**2. Register it with your AI application.**

Claude Code, one command:

```
claude mcp add labview -- C:\Users\Dileep\LabVIEW-AI-Engine-Bridge\bridge.exe -stdio
```

Claude Desktop, edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "labview": {
      "command": "C:\\Users\\Dileep\\LabVIEW-AI-Engine-Bridge\\bridge.exe",
      "args": ["-stdio"]
    }
  }
}
```

Restart the app. You should see nine LabVIEW tools appear.

**3. Prove the chain works before touching LabVIEW.**

```
bridge.exe -mock
```

That starts a fake LabVIEW daemon in-process, opens the dashboard, and prints a
QR code for your phone. Click **build demo VI**. You will see the exact command
stream that LabVIEW will receive. Nothing is real yet, but everything except
LabVIEW is now proven.

**4. Build the addon.** Follow `BRIDGE_VI.md`. It is one VI, about 30-45
minutes, and it is the only hand work in the project. A `.vi` is a binary file,
so unlike a Blender `.py` addon it cannot be shipped as text.

## Use

Open LabVIEW, open `lv_bridge.vi`, press Run, and leave it running. Then talk to
your AI normally:

> Build me a VI with two knobs feeding an Add function, wire the sum to a
> numeric indicator, save it to C:\vis\adder.vi and run it.

The AI calls `create_vi`, then `create_vi_control` twice, `add_block_node`,
`wire_nodes`, `save_vi`, `run_vi`. Watch the front panel assemble itself.

### Tools

| Tool | Purpose |
|---|---|
| `labview_status` | Confirm the bridge VI is running |
| `create_vi` | New blank VI, becomes the active target |
| `create_vi_control` | Place a knob, button, graph, chart, LED, slider, gauge and more |
| `add_block_node` | Place a function or structure on the block diagram |
| `wire_nodes` | Wire two objects by the ids the previous calls returned |
| `set_control_value` | Set a front-panel value |
| `list_objects` | List what has been placed |
| `save_vi` | Save to an absolute path |
| `run_vi` | Run it |

Every creation call returns an id such as `knob1` or `add1`. The AI passes those
to `wire_nodes`. Positions are assigned automatically on a grid, so neither the
AI nor LabVIEW has to think about pixels.

### The dashboard

`bridge.exe` with no flags serves a dark engineering dashboard on port 8080 and
prints a QR code for the LAN address. Scan it from a phone on the same Wi-Fi to
drive the same tools by hand. The session is deliberately ephemeral: no
localStorage, no cookies, no history. Close the tab and the session is gone.

### Flags

```
-stdio                 MCP transport for AI clients
-mock                  fake LabVIEW daemon, for testing without LabVIEW
-port 8080             dashboard port
-labview 127.0.0.1:6060  bridge VI address
-qr-svg lan.svg        also write the QR code as SVG
-no-browser            do not auto-launch the browser
```

## Limits worth knowing

- LabVIEW VI Scripting exists only in the development environment. The IDE has
  to be open and `lv_bridge.vi` has to be running. This cannot ship as a built
  executable.
- The bridge does not read existing VIs. It builds new ones.
- Everything is loopback and LAN only. Nothing leaves the machine.
