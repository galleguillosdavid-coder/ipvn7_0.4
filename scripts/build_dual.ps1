# build_dual.ps1 - Compilacion cruzada dual simultanea (Windows y Linux)
# Ecosistema AFE-Kuzu (SKILL 7.0)

Write-Host "=== INICIANDO COMPILACION CRUZADA DUAL (ipvn7) ===" -ForegroundColor Cyan

$binDir = Join-Path $PSScriptRoot "..\bin"
$winDir = Join-Path $binDir "windows_amd64"
$linDir = Join-Path $binDir "linux_amd64"

New-Item -ItemType Directory -Force -Path $winDir | Out-Null
New-Item -ItemType Directory -Force -Path $linDir | Out-Null

# 1. Compilacion Windows (amd64)
Write-Host "[1/2] Compilando binarios para Windows (.exe)..." -ForegroundColor Yellow
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

go build -ldflags="-s -w" -o "$winDir\ipvn7.exe" .\cmd\ipvn7
go build -ldflags="-s -w" -o "$winDir\ipvn7-cli.exe" .\cmd\ipvn7-cli

Copy-Item "$winDir\ipvn7.exe" "$binDir\ipvn7.exe" -Force
Copy-Item "$winDir\ipvn7-cli.exe" "$binDir\ipvn7-cli.exe" -Force
Write-Host "  -> Generados: bin/windows_amd64/ipvn7.exe y bin/ipvn7.exe" -ForegroundColor Green

# 2. Compilacion Linux (amd64)
Write-Host "[2/2] Compilando binarios para Linux ELF..." -ForegroundColor Yellow
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

go build -ldflags="-s -w" -o "$linDir\ipvn7" .\cmd\ipvn7
go build -ldflags="-s -w" -o "$linDir\ipvn7-cli" .\cmd\ipvn7-cli

Copy-Item "$linDir\ipvn7" "$binDir\ipvn7" -Force
Copy-Item "$linDir\ipvn7-cli" "$binDir\ipvn7-cli" -Force
Write-Host "  -> Generados: bin/linux_amd64/ipvn7 y bin/ipvn7" -ForegroundColor Green

# Restaurar entorno
$env:GOOS = "windows"

Write-Host "[+] Compilacion cruzada dual completada exitosamente!" -ForegroundColor Green
Get-ChildItem -Recurse $binDir | Select-Object Name, Length, LastWriteTime | Format-Table -AutoSize
