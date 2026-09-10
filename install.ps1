<#
.SYNOPSIS
    One-command setup for AI-driven LabVIEW VI generation.

.DESCRIPTION
    Run this once. It does everything:
      1. Verifies Windows, LabVIEW and VI Package Manager are present
      2. Installs Node.js if missing, and refreshes PATH in place so you do not
         have to reopen the terminal
      3. Downloads Jan Goebel's G-AI package from its official GitHub release
      4. Installs it through VI Package Manager
      5. Merges the Claude Desktop config entry, backing up what was there
      6. Restarts LabVIEW and Claude Desktop for you
      7. Waits for the G-AI server to come up and confirms it answers

    Re-running is safe. Every step checks before it acts.

    G-AI is third-party software by Jan Goebel, downloaded from its own release
    and never redistributed by this repo. https://github.com/JanGoebel/G-AI

.EXAMPLE
    .\install.ps1

.EXAMPLE
    .\install.ps1 -NoRestart          # skip the automatic app restarts
#>
[CmdletBinding()]
param(
    [int]    $Port    = 36987,
    [string] $Version = "1.2.4",
    [switch] $NoRestart,
    [switch] $ConfigOnly
)

$ErrorActionPreference = "Stop"
$script:Failed = $false

function Head($t) { Write-Host ""; Write-Host "  $t" -ForegroundColor Cyan; Write-Host ("  " + ("-" * $t.Length)) -ForegroundColor DarkCyan }
function Ok($t)   { Write-Host "  [ok]   $t" -ForegroundColor Green }
function Info($t) { Write-Host "         $t" -ForegroundColor Gray }
function Warn($t) { Write-Host "  [warn] $t" -ForegroundColor Yellow }
function Die($t)  { Write-Host "  [stop] $t" -ForegroundColor Red; $script:Failed = $true }

function Refresh-Path {
    $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' +
                [Environment]::GetEnvironmentVariable('Path', 'User')
}

function Test-Port([int]$p) {
    try {
        $c = New-Object Net.Sockets.TcpClient
        $r = $c.BeginConnect('127.0.0.1', $p, $null, $null)
        $hit = $r.AsyncWaitHandle.WaitOne(500)
        if ($hit) { $c.EndConnect($r) }
        $c.Close()
        return $hit
    } catch { return $false }
}

Write-Host ""
Write-Host "  LabVIEW + AI setup" -ForegroundColor White
Write-Host "  ==================" -ForegroundColor White

# ------------------------------------------------------------- preflight ---
Head "Checking this PC"

if ($env:OS -ne "Windows_NT") { Die "This installer is Windows only."; return }

$lvDirs = @()
foreach ($base in @("$env:ProgramFiles\National Instruments", "${env:ProgramFiles(x86)}\National Instruments")) {
    if (Test-Path $base) { $lvDirs += Get-ChildItem $base -Directory -Filter "LabVIEW*" -ErrorAction SilentlyContinue }
}
if ($lvDirs.Count -gt 0) { Ok "LabVIEW found: $($lvDirs[-1].Name)" }
else { Warn "LabVIEW not found in Program Files. Continuing, but it must be installed." }

$vipm = $null
foreach ($p in @("${env:ProgramFiles(x86)}\JKI\VI Package Manager\vipm.exe",
                 "$env:ProgramFiles\JKI\VI Package Manager\vipm.exe")) {
    if (Test-Path $p) { $vipm = $p; break }
}
if ($vipm) { Ok "VI Package Manager found" }
else { Warn "VI Package Manager not found. It ships with LabVIEW; the .vip will open with whatever is registered." }

# ---------------------------------------------------------------- Node.js ---
if (-not $ConfigOnly) {
    Head "Node.js"
    Refresh-Path
    if (Get-Command npx -ErrorAction SilentlyContinue) {
        Ok "already installed ($(node --version))"
    }
    else {
        Info "not found, installing with winget. This takes a minute."
        if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
            Die "winget is not available. Install Node.js LTS from https://nodejs.org and rerun."
            return
        }
        winget install --id OpenJS.NodeJS.LTS -e --accept-source-agreements --accept-package-agreements | Out-Null
        Refresh-Path
        if (Get-Command npx -ErrorAction SilentlyContinue) { Ok "installed ($(node --version)), PATH refreshed in place" }
        else { Die "Node.js installed but npx is still not on PATH. Reopen PowerShell and rerun this script."; return }
    }
}

# ------------------------------------------------------------ VI package ---
if (-not $ConfigOnly) {
    Head "G-AI VI package"

    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $url = "https://github.com/JanGoebel/G-AI/releases/download/v$Version/jgoebel_lib_g_ai-$Version.0.vip"
    $vip = Join-Path $env:TEMP "jgoebel_lib_g_ai-$Version.0.vip"

    if ((Test-Path $vip) -and ((Get-Item $vip).Length -gt 100000)) {
        Ok "already downloaded"
    }
    else {
        Info "downloading v$Version"
        try {
            Invoke-WebRequest -Uri $url -OutFile $vip -UseBasicParsing
            Ok "downloaded ($([math]::Round((Get-Item $vip).Length / 1MB, 2)) MB)"
        }
        catch {
            Die "download failed: $($_.Exception.Message)"
            Info "Check https://github.com/JanGoebel/G-AI/releases for the current version,"
            Info "then rerun as:  .\install.ps1 -Version <number>"
            return
        }
    }

    Info "opening VI Package Manager"
    Info "Click Install there, accept the dependencies it asks for, then close VIPM."
    Info "This script waits for you."
    try {
        if ($vipm) { Start-Process $vipm -ArgumentList "`"$vip`"" -Wait }
        else       { Start-Process $vip -Wait }
        Ok "VI Package Manager closed"
    }
    catch {
        Die "could not launch VI Package Manager: $($_.Exception.Message)"
        Info "Install it by hand: $vip"
    }
}

# -------------------------------------------------------- Claude Desktop ---
Head "Claude Desktop config"

$dir = Join-Path $env:APPDATA "Claude"
$cfg = Join-Path $dir "claude_desktop_config.json"
if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }

# PowerShell 5.1 has no ConvertFrom-Json -AsHashtable, so convert by hand.
function ConvertTo-Hash($obj) {
    if ($obj -is [System.Management.Automation.PSCustomObject]) {
        $h = @{}
        foreach ($p in $obj.PSObject.Properties) { $h[$p.Name] = ConvertTo-Hash $p.Value }
        return $h
    }
    if ($obj -is [System.Object[]]) { return @($obj | ForEach-Object { ConvertTo-Hash $_ }) }
    return $obj
}

$config = @{}
if (Test-Path $cfg) {
    Copy-Item $cfg "$cfg.bak" -Force
    Ok "backed up existing config to claude_desktop_config.json.bak"
    $raw = Get-Content $cfg -Raw
    if ($raw.Trim()) {
        try { $config = ConvertTo-Hash (ConvertFrom-Json $raw) }
        catch {
            Die "existing config is not valid JSON. Nothing was changed."
            Info "Fix or delete $cfg, then rerun."
            return
        }
    }
}

if (-not $config.ContainsKey("mcpServers")) { $config["mcpServers"] = @{} }
$config["mcpServers"]["G-AI"] = @{
    command = "npx"
    args    = @("mcp-remote", "http://127.0.0.1:$Port/mcp/server")
}
[System.IO.File]::WriteAllText($cfg, (ConvertTo-Json $config -Depth 10), (New-Object System.Text.UTF8Encoding $false))
Ok "config written, servers: $($config['mcpServers'].Keys -join ', ')"

# ------------------------------------------------------------- restarts ---
if (-not $NoRestart) {
    Head "Restarting applications"

    $lvProc = Get-Process -Name LabVIEW -ErrorAction SilentlyContinue
    if ($lvProc) {
        Warn "LabVIEW is running and must restart to pick up the new package."
        Warn "Save your work now."
        $answer = Read-Host "         Close LabVIEW? [y/N]"
        if ($answer -match '^[Yy]') {
            $lvExe = $lvProc[0].Path
            $lvProc | ForEach-Object { $_.CloseMainWindow() | Out-Null }
            Start-Sleep -Seconds 5
            if (Get-Process -Name LabVIEW -ErrorAction SilentlyContinue) {
                Warn "LabVIEW did not close, probably an unsaved-changes dialog. Close it yourself."
            }
            elseif ($lvExe) {
                Start-Process $lvExe
                Ok "LabVIEW restarted"
            }
        }
        else { Info "left running. Restart it yourself before using G-AI." }
    }
    else { Info "LabVIEW is not running. Start it when you are ready." }

    $clProc = Get-Process -Name claude -ErrorAction SilentlyContinue
    $clExe  = if ($clProc) { $clProc[0].Path } else { $null }
    if (-not $clExe) {
        $found = Get-ChildItem (Join-Path $env:LOCALAPPDATA "AnthropicClaude") -Filter "claude.exe" -Recurse -ErrorAction SilentlyContinue |
                 Sort-Object LastWriteTime -Descending | Select-Object -First 1
        if ($found) { $clExe = $found.FullName }
    }
    if ($clProc) {
        $clProc | Stop-Process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 2
        Ok "Claude Desktop closed, including the tray process"
    }
    if ($clExe) { Start-Process $clExe; Ok "Claude Desktop restarted" }
    else { Warn "Could not locate claude.exe. Start Claude Desktop yourself." }
}

# ------------------------------------------------------------- verify ---
Head "Verifying"

if (Test-Port $Port) {
    Ok "something is listening on port $Port"
    Ok "Setup complete. Ask Claude Desktop to build you a VI."
}
else {
    Info "Port $Port is quiet, which is expected until you launch the server."
    Write-Host ""
    Write-Host "  Two clicks left:" -ForegroundColor Cyan
    Write-Host "    1. In LabVIEW, open  Tools > G-AI  and launch it. Leave it running."
    Write-Host "    2. In Claude Desktop, start a new chat."
    Write-Host ""
    $wait = Read-Host "         Press Enter once G-AI is running, or S to skip"
    if ($wait -notmatch '^[Ss]') {
        $tries = 0
        while ($tries -lt 20 -and -not (Test-Port $Port)) { Start-Sleep -Seconds 1; $tries++ }
        if (Test-Port $Port) { Ok "G-AI is answering on port $Port" }
        else { Warn "Still nothing on port $Port. Check the Tools menu in LabVIEW." }
    }
}

Write-Host ""
if ($script:Failed) {
    Write-Host "  Finished with problems above." -ForegroundColor Yellow
} else {
    Write-Host "  Try this in Claude Desktop:" -ForegroundColor Cyan
    Write-Host '    "In LabVIEW, create a VI that adds two numbers and shows the result."'
    Write-Host ""
    Write-Host "  Check it registered: File > Settings > Developer should list G-AI." -ForegroundColor DarkGray
}
Write-Host ""
