# Install

One command does everything. Windows, LabVIEW, Claude Desktop.

## The whole install

Open **PowerShell** and run these three lines:

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force
git clone https://github.com/dileep12072024-sudo/labview-ai-engine-bridge.git C:\LabVIEW-AI-Engine-Bridge
cd C:\LabVIEW-AI-Engine-Bridge; .\install.ps1
```

That is the install. There is no second thing to install afterwards.

If `git` is missing, get it first with `winget install --id Git.Git -e`, then
close and reopen PowerShell.

## What the script does

1. Checks that LabVIEW and VI Package Manager are on the PC.
2. Installs Node.js if missing, and refreshes PATH in place, so you never have
   to close and reopen the terminal.
3. Downloads the G-AI package from its official GitHub release.
4. Opens VI Package Manager and waits while you click **Install**.
5. Merges the Claude Desktop config entry, backing up your existing file and
   leaving any other MCP servers you have untouched.
6. Offers to close and restart LabVIEW, asking first so you can save your work.
7. Closes and restarts Claude Desktop, including the tray process that people
   normally forget.
8. Waits for the server to come up and confirms it is answering.

Re-running it is safe. Every step checks before it acts.

## The two clicks it cannot do for you

VI Package Manager needs you to press **Install** in its window, and LabVIEW
needs you to launch **Tools ▸ G-AI** once. The script pauses and waits at both
points, then verifies the result.

## Then use it

Leave G-AI running in LabVIEW. In Claude Desktop, ask:

> In LabVIEW, create a VI that adds two numbers and shows the result.

Start small and confirm it works before asking for a PID controller.

## Options

```powershell
.\install.ps1 -NoRestart      # do not touch LabVIEW or Claude Desktop
.\install.ps1 -ConfigOnly     # only rewrite the Claude Desktop config
.\install.ps1 -Version 1.2.5  # pin a different G-AI release
.\install.ps1 -Port 40000     # use a different port
```

## Troubleshooting

| Symptom | Fix |
|---|---|
| `install.ps1 cannot be loaded` | Run the `Set-ExecutionPolicy` line above first |
| Tools do not appear in Claude Desktop | Quit from the system tray, not the window X, then reopen |
| G-AI missing from the LabVIEW Tools menu | The VIPM install did not finish. Rerun the script |
| Download fails | Check the release list, rerun with `-Version <number>` |
| Nothing listening on the port | Launch `Tools ▸ G-AI` in LabVIEW |
| Generation is patchy on complex VIs | Known. G-AI's author calls code generation not yet feature complete |

## What is actually being installed

[G-AI](https://github.com/JanGoebel/G-AI) by Jan Goebel, a LabVIEW MCP server
that runs inside LabVIEW. It is downloaded from its own GitHub release at
install time, never redistributed by this repo. `FAST_PATH.md` has the manual
steps and a deeper alternative if you want them.

## Optional extras in this repo

Neither is needed for VI generation.

**Phone dashboard.** `dist\bridge.exe` is a Go server that puts an ephemeral
control dashboard on your phone over Wi-Fi via a QR code, with no cookies and no
stored state. It talks to a different LabVIEW receiver, `lv_bridge.vi`, which
does not exist yet. `BRIDGE_VI.md` explains how to build it.

**Offline MCP server.** The same binary speaks MCP over stdio with no Node.js at
all. Same caveat: it needs `lv_bridge.vi`.
