param(
    [string]$ToolkitDir = "C:\LabVIEW-MCP-Server-Toolkit"
)

Write-Host "Checking prerequisites..." -ForegroundColor Cyan

# 1. Check Node/NPM
if (-not (Get-Command "npx" -ErrorAction SilentlyContinue)) {
    Write-Host "Node.js (npx) is not installed. Installing automatically via winget..." -ForegroundColor Yellow
    winget install --id OpenJS.NodeJS.LTS -e --silent --accept-package-agreements --accept-source-agreements
    
    # Reload environment variables for current process
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
    if (-not (Get-Command "npx" -ErrorAction SilentlyContinue)) {
        Write-Host "Node.js installed, but you may need to close and reopen PowerShell for it to be recognized." -ForegroundColor Red
        exit 1
    }
}

# 2. Download or Update Toolkit (No Git required)
if (Test-Path $ToolkitDir) {
    Write-Host "LabVIEW-MCP-Server-Toolkit already exists at $ToolkitDir." -ForegroundColor Cyan
} else {
    Write-Host "Downloading LabVIEW-MCP-Server-Toolkit to $ToolkitDir..." -ForegroundColor Cyan
    $ZipPath = "$env:TEMP\toolkit.zip"
    Invoke-WebRequest -Uri "https://github.com/JanGoebel/LabVIEW-MCP-Server-Toolkit/archive/refs/heads/main.zip" -OutFile $ZipPath
    
    Write-Host "Extracting toolkit..." -ForegroundColor Cyan
    Expand-Archive -Path $ZipPath -DestinationPath $env:TEMP -Force
    Move-Item -Path "$env:TEMP\LabVIEW-MCP-Server-Toolkit-main" -Destination $ToolkitDir -Force
    Remove-Item $ZipPath -Force
}

# 3. Check for VIPM Dependencies
Write-Host ""
Write-Host "IMPORTANT: Please ensure you have the following VIPM dependencies installed in LabVIEW:" -ForegroundColor Yellow
Write-Host " - IG HTTP Server Toolkit" -ForegroundColor Yellow
Write-Host " - JKI JSONtext" -ForegroundColor Yellow
Write-Host ""

# 4. Configure Claude Desktop
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
    "command" = "npx.cmd"
    "args" = @("-y", "mcp-remote", "http://127.0.0.1:36987/mcp/server")
}

$config | ConvertTo-Json -Depth 10 | Set-Content $ClaudeConfigPath
Write-Host "Added 'LabVIEW-Assistant' MCP server to Claude Desktop config at $ClaudeConfigPath." -ForegroundColor Green

Write-Host "--------------------------------------------------------" -ForegroundColor Cyan
Write-Host "INSTALLATION COMPLETE!" -ForegroundColor Green
Write-Host "Next Steps:"
Write-Host "1. Restart Claude Desktop (fully quit from the system tray)."
Write-Host "2. Open 'C:\Users\Dileep\LabVIEW-AI-Engine-Bridge\VI Scripting Server.lvproj' in LabVIEW 2025."
Write-Host "3. Run 'Scripting Server\Main.vi' in LabVIEW."
Write-Host "4. Start chatting with Claude!"
