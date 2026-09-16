param(
    [string]$ToolkitDir = "C:\LabVIEW-MCP-Server-Toolkit"
)

Write-Host "Checking prerequisites..." -ForegroundColor Cyan

# 1. Check Git
if (-not (Get-Command "git" -ErrorAction SilentlyContinue)) {
    Write-Host "Git is not installed. Please install Git (e.g. winget install Git.Git)." -ForegroundColor Red
    exit 1
}

# 2. Check Node/NPM
if (-not (Get-Command "npx" -ErrorAction SilentlyContinue)) {
    Write-Host "Node.js (npx) is not installed. Please install Node.js (e.g. winget install OpenJS.NodeJS.LTS)." -ForegroundColor Red
    exit 1
}

# 3. Clone or Update Toolkit
if (Test-Path $ToolkitDir) {
    Write-Host "Updating LabVIEW-MCP-Server-Toolkit in $ToolkitDir..." -ForegroundColor Cyan
    Push-Location $ToolkitDir
    git pull
    Pop-Location
} else {
    Write-Host "Cloning LabVIEW-MCP-Server-Toolkit to $ToolkitDir..." -ForegroundColor Cyan
    git clone https://github.com/JanGoebel/LabVIEW-MCP-Server-Toolkit.git $ToolkitDir
}

# 4. Check for VIPM Dependencies
Write-Host ""
Write-Host "IMPORTANT: Please ensure you have the following VIPM dependencies installed in LabVIEW:" -ForegroundColor Yellow
Write-Host " - IG HTTP Server Toolkit" -ForegroundColor Yellow
Write-Host " - JKI JSONtext" -ForegroundColor Yellow
Write-Host ""

# 5. Configure Claude Desktop
$ClaudeConfigDir = "$env:APPDATA\Claude"
$ClaudeConfigPath = "$ClaudeConfigDir\claude_desktop_config.json"

if (-not (Test-Path $ClaudeConfigDir)) {
    New-Item -ItemType Directory -Force -Path $ClaudeConfigDir | Out-Null
}

$config = @{}
if (Test-Path $ClaudeConfigPath) {
    $configJson = Get-Content $ClaudeConfigPath -Raw
    try {
        $config = $configJson | ConvertFrom-Json -AsHashtable
    } catch {
        Write-Host "Could not parse existing claude_desktop_config.json. Proceeding with new config." -ForegroundColor Yellow
    }
}

if (-not $config.ContainsKey("mcpServers")) {
    $config["mcpServers"] = @{}
}

$config["mcpServers"]["LabVIEW-Assistant"] = @{
    "command" = "npx"
    "args" = @("mcp-remote", "http://127.0.0.1:36987/mcp/server")
}

$config | ConvertTo-Json -Depth 10 | Set-Content $ClaudeConfigPath
Write-Host "Added 'LabVIEW-Assistant' MCP server to Claude Desktop config at $ClaudeConfigPath." -ForegroundColor Green

Write-Host "--------------------------------------------------------" -ForegroundColor Cyan
Write-Host "INSTALLATION COMPLETE!" -ForegroundColor Green
Write-Host "Next Steps:"
Write-Host "1. Restart Claude Desktop (fully quit from the system tray)."
Write-Host "2. Open '$PSScriptRoot\VI Scripting Server.lvproj' in LabVIEW 2025 (version 25.0)."
Write-Host "3. Run 'Scripting Server\Main.vi' in LabVIEW."
Write-Host "4. Start chatting with Claude!"
