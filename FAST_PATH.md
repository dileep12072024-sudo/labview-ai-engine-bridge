# Fast path: skip building the VI yourself

Someone has already built and packaged the LabVIEW side. Jan Goebel's **G-AI**
is a LabVIEW MCP server that installs as a single VI package and puts a launcher
in your Tools menu. No hand-wiring, no `lv_bridge.vi`.

If your goal is "make the AI build VIs today", start here. Come back to
`BRIDGE_VI.md` only if you want the phone dashboard and the ephemeral session
this repo adds.

- Project: https://github.com/JanGoebel/G-AI
- Licence: MIT
- Status: the author calls code generation "not at all feature complete". It
  reads projects and block diagrams well, and creates VIs and adds code with
  gaps. It is third-party software, not mine and not audited by me.

---

## Just run the installer

`INSTALL.md` has the one-command version. `.\install.ps1` does everything below
automatically, including the restarts and a final check that the server answers.

The manual steps that follow are for when you would rather do it yourself, or
when something goes wrong and you need to see the pieces.

---

## Step 1. Install Node.js

The MCP server runs inside LabVIEW over HTTP, and Claude Desktop reaches it
through `mcp-remote`, which needs Node.

```powershell
winget install --id OpenJS.NodeJS.LTS -e
```

Close and reopen PowerShell, then confirm:

```powershell
node --version
npx --version
```

## Step 2. Download the VI package

```powershell
curl.exe -L -o "$env:USERPROFILE\Downloads\g-ai.vip" https://github.com/JanGoebel/G-AI/releases/download/v1.2.4/jgoebel_lib_g_ai-1.2.4.0.vip
```

## Step 3. Install it

Double-click the downloaded `g-ai.vip`. VI Package Manager opens. It ships with
LabVIEW, so it should already be there. Click **Install**.

It pulls two dependencies automatically. If it asks, say yes to both:

- IG HTTP Server Toolkit
- JKI JSONtext

Restart LabVIEW. You should now see **G-AI** under the `Tools` menu.

## Step 4. Start the server

In LabVIEW: `Tools ▸ G-AI`. Launch it. It opens an HTTP server on port `36987`.
Leave it running, exactly like leaving the Blender addon enabled.

## Step 5. Point Claude Desktop at it

```powershell
notepad $env:APPDATA\Claude\claude_desktop_config.json
```

Paste this. If the file already has servers, add only the `"G-AI"` block inside
the existing `mcpServers` object.

```json
{
  "mcpServers": {
    "G-AI": {
      "command": "npx",
      "args": ["mcp-remote", "http://127.0.0.1:36987/mcp/server"]
    }
  }
}
```

Quit Claude Desktop from the system tray, not the window X, then reopen it.

**Verify:** `File ▸ Settings ▸ Developer` should list G-AI as running. In a new
chat, the `+` icon under the message box shows it under Connectors.

## Step 6. Ask for a VI

> In LabVIEW, create a VI that generates a random number between a min and a max
> value, wire it up, and save it to C:\vis\random.vi

---

## If G-AI's code generation is too thin

The same author maintains **labview_assistant**, which carries a much deeper
scripting API: `create_control`, `connect_objects` for wiring, `enclose_selection`
for wrapping code in loops, `add_subvi`, `connect_to_pane`, `cleanup_vi` for
auto-layout, and `get_vi_frontpanel_picture` so the AI can look at what it built.
That last one is the Blender viewport-screenshot equivalent.

It is not packaged, so you clone and run it:

```powershell
git clone https://github.com/JanGoebel/labview_assistant.git C:\labview_assistant
```

Open `C:\labview_assistant\VI Scripting Server.lvproj` in LabVIEW, then open and
run `Scripting Server\Main.vi`. It needs the same two VIPM dependencies as
above, plus the LabVIEW MCP Server Toolkit:

```powershell
git clone https://github.com/JanGoebel/LabVIEW-MCP-Server-Toolkit.git C:\LabVIEW-MCP-Server-Toolkit
```

Claude Desktop config is the same `mcp-remote` block, same port.

Note: `labview_assistant` has no licence file, so its terms are unstated. The
toolkit it depends on is MIT.

---

## How this relates to this repo

| | This repo | G-AI |
|---|---|---|
| LabVIEW side | You build `lv_bridge.vi` once | Prebuilt, installs in one click |
| Phone dashboard over Wi-Fi with QR | Yes | No |
| Ephemeral session, no cookies or storage | Yes | Not applicable |
| Runs offline with no Node.js | Yes, single Go binary | Needs Node for `mcp-remote` |
| Ready to build VIs today | No | Yes |

Use G-AI to start working now. This repo remains the offline, phone-accessible
option if you decide you want that later.
