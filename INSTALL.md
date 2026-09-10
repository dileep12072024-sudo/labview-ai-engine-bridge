# Install on a new PC

> **Want it working today without building a VI by hand?** Run `.\install.ps1`
> and read `FAST_PATH.md`. It installs Jan Goebel's G-AI, a prebuilt LabVIEW MCP
> server, and wires it into Claude Desktop. This repo's own path needs
> `lv_bridge.vi` built once, and adds a phone dashboard in return.

Windows, with Claude Desktop. Every command below is copy-paste ready. Run them
in **PowerShell**, not Command Prompt.

Total time: about 5 minutes, plus the LabVIEW side.

---

## Step 1. Get the files

Install the GitHub CLI if the PC does not have it:

```powershell
winget install --id GitHub.cli -e
```

Close and reopen PowerShell, then sign in. This opens a browser once:

```powershell
gh auth login --web --git-protocol https
```

Clone the repo to the root of C:

```powershell
git clone https://github.com/dileep12072024-sudo/labview-ai-engine-bridge.git C:\LabVIEW-AI-Engine-Bridge
cd C:\LabVIEW-AI-Engine-Bridge
```

The Windows binary is already in the clone at `dist\bridge.exe`. Nothing to
build and no Go toolchain needed.

---

## Step 2. Check the binary runs

```powershell
C:\LabVIEW-AI-Engine-Bridge\dist\bridge.exe -mock -no-browser
```

You should see a QR code, your LAN address, and `mock LabVIEW daemon listening`.
Open `http://localhost:8080` in a browser and click **build demo VI**. The log
pane fills with commands.

Press `Ctrl+C` to stop it. That single test proves the server, the dashboard, the
WebSocket and the command translation all work. LabVIEW is not involved yet.

---

## Step 3. Connect it to Claude Desktop

Create or edit the config file:

```powershell
notepad $env:APPDATA\Claude\claude_desktop_config.json
```

Paste this. If the file already has other servers, add only the `"labview"`
block inside the existing `mcpServers` object.

```json
{
  "mcpServers": {
    "labview": {
      "command": "C:\\LabVIEW-AI-Engine-Bridge\\dist\\bridge.exe",
      "args": ["-stdio"]
    }
  }
}
```

Save, then fully quit Claude Desktop from the system tray and reopen it.

**Verify:** click the tools icon in the message box. You should see nine tools
whose names start with `create_vi`, `add_block_node`, `wire_nodes` and so on.

For Claude Code instead of Claude Desktop, this one command replaces the whole
step:

```powershell
claude mcp add labview -- C:\LabVIEW-AI-Engine-Bridge\dist\bridge.exe -stdio
```

---

## Step 4. Build the LabVIEW addon

Open `BRIDGE_VI.md` and follow it. Nine phases, roughly 30 to 45 minutes, with a
test command after each one. This is the only hand work in the project, and it
is required. Without it LabVIEW has nothing listening.

Test any single phase like this, while `lv_bridge.vi` is running:

```powershell
C:\LabVIEW-AI-Engine-Bridge\dist\bridge.exe -send "CMD:PING"
```

---

## Step 5. Use it

1. Open LabVIEW.
2. Open `C:\LabVIEW-AI-Engine-Bridge\lv_bridge.vi`.
3. Press the Run arrow and leave it running.
4. In Claude Desktop, ask for what you want:

> Build a VI with two knobs feeding an Add function, wire the sum to a numeric
> indicator, save it to C:\vis\adder.vi and run it.

Watch the front panel assemble itself.

---

## Optional: control it from your phone

Start the dashboard instead of the stdio server:

```powershell
C:\LabVIEW-AI-Engine-Bridge\dist\bridge.exe
```

It prints a QR code. Scan it with a phone on the same Wi-Fi and you get the same
tools as buttons. The session is deliberately ephemeral, with no cookies, no
localStorage and no history. Close the tab and it is gone.

---

## Rebuilding from source

Only needed if you change the Go code.

```powershell
winget install --id GoLang.Go -e
cd C:\LabVIEW-AI-Engine-Bridge
go mod tidy
go test ./...
go build -o dist\bridge.exe .
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Tools do not appear in Claude Desktop | Config not loaded | Quit from the system tray, not the window X, then reopen |
| `LabVIEW bridge VI unreachable` | `lv_bridge.vi` is not running | Open it in LabVIEW and press Run |
| `-send` hangs forever | TCP Read mode is not CRLF | See Phase 1 of `BRIDGE_VI.md` |
| Port 8080 already in use | Something else has it | Add `-port 8090` |
| Scripting methods missing in LabVIEW | Scripting not enabled | `Tools ▸ Options ▸ VI Server`, tick Show VI Scripting operations, restart |
