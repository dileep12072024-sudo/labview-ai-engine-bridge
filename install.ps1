<#
.SYNOPSIS
    Sets up AI-driven LabVIEW VI generation on this PC.

.DESCRIPTION
    Installs the pieces needed for Claude Desktop to build LabVIEW VIs:
      1. Node.js, which Claude Desktop needs to reach an HTTP MCP server
      2. Jan Goebel's G-AI VI package, downloaded from its official release
      3. The Claude Desktop config entry, merged without disturbing other servers

    G-AI is third-party MIT-spirited software by Jan Goebel. It is downloaded
    from its GitHub release, never redistributed by this repo.
    https://github.com/JanGoebel/G-AI

.EXAMPLE
    .\install.ps1

.EXAMPLE
    .\install.ps1 -Port 36987 -SkipNode
#>
[CmdletBinding()]
param(
    [int]    $Port = 36987,
    [string] $Version = "1.2.4",
    [switch] $SkipNode,
    [switch] $ConfigOnly
)

$ErrorActionPreference = "Stop"

function Say([string]$msg, [string]$colour = "White") { Write-Host $msg -ForegroundColor $colour }
function Step([string]$msg) { Write-Host ""; Write-Host "==> $msg" -ForegroundColor Cyan }

Say ""
Say "LabVIEW AI setup" Cyan
Say "================" Cyan

# ---------------------------------------------------------------- Node.js ---
if (-not $ConfigOnly -and -not $SkipNode) {
    Step "Checking Node.js"
    if (Get-Command npx -ErrorAction SilentlyContinue) {
        Say "    found $(node --version)" Green
    }
    else {
        Say "    not found, installing via winget" Yellow
        winget install --id OpenJS.NodeJS.LTS -e --accept-source-agreements --accept-package-agreements
        Say ""
        Say "    Node.js installed. CLOSE THIS WINDOW, open a new PowerShell," Yellow
        Say "    and run this script again so npx is on your PATH." Yellow
        return
    }
}

# ------------------------------------------------------------ VI package ---
if (-not $ConfigOnly) {
    Step "Downloading the G-AI VI package"
    # Windows PowerShell 5.1 can default to TLS 1.0, which GitHub refuses.
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $url = "https://github.com/JanGoebel/G-AI/releases/download/v$Version/jgoebel_lib_g_ai-$Version.0.vip"
    $vip = Join-Path $env:USERPROFILE "Downloads\jgoebel_lib_g_ai-$Version.0.vip"

    if (Test-Path $vip) {
        Say "    already downloaded: $vip" Green
    }
    else {
        Say "    $url"
        try {
            Invoke-WebRequest -Uri $url -OutFile $vip -UseBasicParsing
            Say "    saved to $vip" Green
        }
        catch {
            Say "    download failed: $($_.Exception.Message)" Red
            Say "    Check https://github.com/JanGoebel/G-AI/releases for the current version," Yellow
            Say "    then rerun with -Version <number>." Yellow
            return
        }
    }

    Step "Installing it"
    Say "    Opening the package in VI Package Manager. Click Install there,"
    Say "    accept the two dependencies it asks for, then restart LabVIEW."
    Start-Process $vip
}

# -------------------------------------------------------- Claude Desktop ---
Step "Configuring Claude Desktop"

$dir = Join-Path $env:APPDATA "Claude"
$cfg = Join-Path $dir "claude_desktop_config.json"
if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }

# PowerShell 5.1 has no ConvertFrom-Json -AsHashtable, so convert by hand.
function ToHashtable($obj) {
    if ($obj -is [System.Management.Automation.PSCustomObject]) {
        $h = @{}
        foreach ($p in $obj.PSObject.Properties) { $h[$p.Name] = ToHashtable $p.Value }
        return $h
    }
    if ($obj -is [System.Object[]]) { return @($obj | ForEach-Object { ToHashtable $_ }) }
    return $obj
}

$config = @{}
if (Test-Path $cfg) {
    $backup = "$cfg.bak"
    Copy-Item $cfg $backup -Force
    Say "    backed up existing config to $backup" Green
    $raw = Get-Content $cfg -Raw
    if ($raw.Trim()) {
        try { $config = ToHashtable (ConvertFrom-Json $raw) }
        catch {
            Say "    existing config is not valid JSON, aborting so nothing is lost" Red
            Say "    fix or delete $cfg, then rerun" Yellow
            return
        }
    }
}

if (-not $config.ContainsKey("mcpServers")) { $config["mcpServers"] = @{} }
$config["mcpServers"]["G-AI"] = @{
    command = "npx"
    args    = @("mcp-remote", "http://127.0.0.1:$Port/mcp/server")
}

$json = ConvertTo-Json $config -Depth 10
[System.IO.File]::WriteAllText($cfg, $json, (New-Object System.Text.UTF8Encoding $false))
Say "    wrote $cfg" Green
Say "    servers now configured: $($config['mcpServers'].Keys -join ', ')" Green

# ------------------------------------------------------------- Next steps ---
Say ""
Say "Done. Four things left, in order:" Cyan
Say ""
Say "  1. Finish the VI Package Manager install if it is still open."
Say "  2. Restart LabVIEW. You should see G-AI under the Tools menu."
Say "  3. Launch it from Tools. It listens on port $Port. Leave it running."
Say "  4. Quit Claude Desktop from the SYSTEM TRAY, not the window X, then reopen."
Say ""
Say "Then ask Claude Desktop:" Cyan
Say '  "In LabVIEW, create a VI that adds two numbers and shows the result."'
Say ""
Say "Verify in Claude Desktop: File > Settings > Developer should list G-AI as running." DarkGray
Say ""
