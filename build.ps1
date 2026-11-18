# build.ps1 — Windows 下构建 uniTerm
# 用法（在项目根目录）：
#   powershell -ExecutionPolicy Bypass -File build.ps1
#   powershell -ExecutionPolicy Bypass -File build.ps1 -Compress
# 产物：bin\uniTerm.exe

param([switch]$Compress)

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

$upx = $null
if ($Compress) {
    $upx = (Get-Command upx.exe -CommandType Application -ErrorAction Stop).Source
}

# 1. 确保 wails3 可用（GOPATH\bin 可能不在 PATH 里）
$gobin = Join-Path (go env GOPATH) 'bin'
if (-not (Get-Command wails3 -ErrorAction SilentlyContinue)) {
    $env:PATH = "$gobin;$env:PATH"
}
if (-not (Get-Command wails3 -ErrorAction SilentlyContinue)) {
    Write-Host '未找到 wails3，正在安装 v3.0.0-beta.12 ...'
    go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.12
    $env:PATH = "$gobin;$env:PATH"
}

# 2. 安装前端依赖（已装过会很快）
Push-Location frontend
npm install
Pop-Location

# 3. 生产构建
wails3 build
if ($LASTEXITCODE -ne 0) {
    throw "Wails build failed (exit $LASTEXITCODE)."
}

if ($Compress) {
    & $upx --best --lzma bin\uniTerm.exe
    if ($LASTEXITCODE -ne 0) {
        throw "UPX compression failed (exit $LASTEXITCODE)."
    }
    & $upx -t bin\uniTerm.exe
    if ($LASTEXITCODE -ne 0) {
        throw "UPX integrity test failed (exit $LASTEXITCODE)."
    }
}

Write-Host ''
Write-Host '构建完成：bin\uniTerm.exe'
