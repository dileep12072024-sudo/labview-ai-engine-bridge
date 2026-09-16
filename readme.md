# LabVIEW-AI-Engine-Bridge (Powered by LabVIEW Assistant)

This repository is a fully functional LabVIEW MCP (Model Context Protocol) Server. It acts as the "Hands" for an AI assistant like Claude, allowing the AI to programmatically generate LabVIEW code, drop components, wire blocks, and even get snapshots of the front panel.

This is a customized setup built on top of [JanGoebel's labview_assistant](https://github.com/JanGoebel/labview_assistant), designed for an autonomous closed-loop AI workflow (e.g., Claude + Screen Sharing MCP + LabVIEW MCP).

## Minimum Requirements
- **LabVIEW 2025 (Version 25.0)** or newer (the `.lvproj` is saved in this format)
- **Node.js** (for `mcp-remote`)
- **Git**

## One-Click Installation

We have provided a smart PowerShell installer that automatically:
1. Checks for Git and Node.js.
2. Clones the necessary `LabVIEW-MCP-Server-Toolkit` dependency.
3. Automatically injects the LabVIEW MCP server into your Claude Desktop configuration file.

Open PowerShell as Administrator and run:
```powershell
Invoke-RestMethod -Uri "https://raw.githubusercontent.com/dileep12072024-sudo/labview-ai-engine-bridge/main/install.ps1" | Invoke-Expression
```

*Note: You must still ensure you have `IG HTTP Server Toolkit` and `JKI JSONtext` installed via VIPM (VI Package Manager).*

## Running the Server

1. **Restart Claude Desktop** (Quit completely from the system tray, then reopen it).
2. **Open LabVIEW 2025**.
3. Open `VI Scripting Server.lvproj` located in this repository.
4. Run `Scripting Server\Main.vi`.

You are now ready to chat with Claude and ask it to build VIs for you!
