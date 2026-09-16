param(
    [string]$GAIDownloadUrl = "https://github.com/JanGoebel/G-AI/archive/refs/heads/main.zip",
    [string]$GAIDir = "C:\G-AI"
)

Write-Host "==============================================" -ForegroundColor Cyan
Write-Host " G-AI & Screen Eyes MCP Auto-Installer" -ForegroundColor Cyan
Write-Host "==============================================" -ForegroundColor Cyan

# 1. Check Node/NPM
if (-not (Get-Command "npx" -ErrorAction SilentlyContinue)) {
    Write-Host "Node.js (npx) is not installed. Installing automatically via winget..." -ForegroundColor Yellow
    winget install --id OpenJS.NodeJS.LTS -e --silent --accept-package-agreements --accept-source-agreements
    
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}

# 2. Download G-AI
if (-not (Test-Path $GAIDir)) {
    Write-Host "Downloading G-AI to $GAIDir..." -ForegroundColor Cyan
    $ZipPath = "$env:TEMP\g-ai.zip"
    Invoke-WebRequest -Uri $GAIDownloadUrl -OutFile $ZipPath
    Expand-Archive -Path $ZipPath -DestinationPath $env:TEMP -Force
    Move-Item -Path "$env:TEMP\G-AI-main" -Destination $GAIDir -Force
    Remove-Item $ZipPath -Force
}

# 3. Configure Claude Desktop
$ClaudeConfigDir = "$env:APPDATA\Claude"
$ClaudeConfigPath = "$ClaudeConfigDir\claude_desktop_config.json"
if (-not (Test-Path $ClaudeConfigDir)) { New-Item -ItemType Directory -Force -Path $ClaudeConfigDir | Out-Null }

$config = @{}
if (Test-Path $ClaudeConfigPath) {
    try { $config = (Get-Content $ClaudeConfigPath -Raw) | ConvertFrom-Json -AsHashtable } catch { }
}
if (-not $config.ContainsKey("mcpServers")) { $config["mcpServers"] = @{} }

# Add G-AI MCP
$config["mcpServers"]["G-AI-LabVIEW"] = @{
    "command" = "npx.cmd"
    "args" = @("-y", "mcp-remote", "http://127.0.0.1:36987/mcp/server")
}

# Add Screen Sharing / Eyes MCP
$config["mcpServers"]["Screen-Eyes"] = @{
    "command" = "npx.cmd"
    "args" = @("-y", "@modelcontextprotocol/server-puppeteer") # Replace with your preferred screen capture MCP package
}

$config | ConvertTo-Json -Depth 10 | Set-Content $ClaudeConfigPath
Write-Host "Configured G-AI and Screen-Eyes MCP servers in Claude Desktop." -ForegroundColor Green

# 4. Output Prompts/Commands for Codex and AGY
Write-Host ""
Write-Host "==============================================" -ForegroundColor Yellow
Write-Host " CODEX & AGY (ANTIGRAVITY) SETUP INSTRUCTIONS" -ForegroundColor Yellow
Write-Host "==============================================" -ForegroundColor Yellow
Write-Host ""
Write-Host "To install these two MCPs in Codex, run this in Codex terminal:"
Write-Host "  > cursor-mcp add G-AI-LabVIEW npx.cmd -y mcp-remote http://127.0.0.1:36987/mcp/server"
Write-Host "  > cursor-mcp add Screen-Eyes npx.cmd -y @modelcontextprotocol/server-puppeteer"
Write-Host ""
Write-Host "To install these two MCPs in AGY (Antigravity), run this in WSL/AGY:"
Write-Host '  > cat <<EOF > ~/.gemini/config/mcp_config.json'
Write-Host '    {'
Write-Host '      "mcpServers": {'
Write-Host '        "G-AI-LabVIEW": { "command": "npx", "args": ["-y", "mcp-remote", "http://127.0.0.1:36987/mcp/server"] },'
Write-Host '        "Screen-Eyes": { "command": "npx", "args": ["-y", "@modelcontextprotocol/server-puppeteer"] }'
Write-Host '      }'
Write-Host '    }'
Write-Host '    EOF'
Write-Host ""
Write-Host "==============================================" -ForegroundColor Cyan
Write-Host " VERSION REQUIREMENTS" -ForegroundColor Cyan
Write-Host " - LabVIEW: Version 2025 or newer"
Write-Host " - Node.js: Version 18.x or newer (for npx)"
Write-Host " - VIPM Dependencies: 'IG HTTP Server Toolkit' & 'JKI JSONtext'"
Write-Host "==============================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next steps for G-AI:"
Write-Host "1. Open LabVIEW and install the VIP package from C:\G-AI\builds\G-AI"
Write-Host "2. Go to Tools -> G-AI to launch the server."
Write-Host "3. Open Claude/Codex/AGY and start chatting!"
