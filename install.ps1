# ==============================================================================
# ipvn7 Network OS — Instalador Universal Soberano de 1 Clic (Windows)
# Repositorio: https://github.com/galleguillosdavid-coder/ipvn7_0.4
# ==============================================================================

param(
    [string]$InstallDir = "C:\ipvn7",
    [int]$P2PPort = 7777,
    [int]$WebPort = 7070,
    [switch]$StartService
)

$ErrorActionPreference = "Stop"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   ipvn7 Network OS — Instalación Soberana (Windows)     " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Crear directorios
Write-Host "[1/4] Creando directorios en $InstallDir..." -ForegroundColor Yellow
$dirs = @(
    "$InstallDir\bin",
    "$InstallDir\keystore",
    "$InstallDir\logs",
    "$InstallDir\web"
)
foreach ($d in $dirs) {
    if (-not (Test-Path $d)) {
        New-Item -ItemType Directory -Path $d -Force | Out-Null
    }
}

# 2. Copiar o Descargar Binarios
Write-Host "[2/4] Instalando binarios del nodo..." -ForegroundColor Yellow
if (Test-Path ".\bin\windows_amd64\ipvn7.exe") {
    Copy-Item ".\bin\windows_amd64\ipvn7.exe" "$InstallDir\bin\ipvn7.exe" -Force
    Copy-Item ".\bin\windows_amd64\ipvn7-cli.exe" "$InstallDir\bin\ipvn7-cli.exe" -Force
    if (Test-Path ".\web") {
        Copy-Item -Recurse -Path ".\web\*" -Destination "$InstallDir\web\" -Force
    }
} else {
    $zipUrl = "https://github.com/galleguillosdavid-coder/ipvn7_0.4/releases/latest/download/ipvn7-v0.4.0-windows-amd64.zip"
    $tmpZip = "$env:TEMP\ipvn7.zip"
    Write-Host "[+] Descargando release desde GitHub..." -ForegroundColor Yellow
    try {
        Invoke-WebRequest -Uri $zipUrl -OutFile $tmpZip -UseBasicParsing
        Expand-Archive -Path $tmpZip -DestinationPath $InstallDir -Force
        Remove-Item $tmpZip -Force
    } catch {
        Write-Host "[!] Compilando localmente con Go..." -ForegroundColor Yellow
        go build -o "$InstallDir\bin\ipvn7.exe" ./cmd/ipvn7
        go build -o "$InstallDir\bin\ipvn7-cli.exe" ./cmd/ipvn7-cli
        Copy-Item -Recurse -Path ".\web\*" -Destination "$InstallDir\web\" -Force
    }
}

# 3. Registrar en PATH del Sistema
Write-Host "[3/4] Agregando a variable de entorno PATH..." -ForegroundColor Yellow
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notmatch [regex]::Escape("$InstallDir\bin")) {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir\bin", "User")
    Write-Host "[+] Directorio bin agregado al PATH de usuario." -ForegroundColor Green
}

# Crear script de arranque persistente
$startBat = @"
@echo off
title ipvn7 Network OS Daemon
cd /d "$InstallDir"
"$InstallDir\bin\ipvn7.exe" --port $P2PPort --web-port $WebPort --keystore "$InstallDir\keystore\node_identity.key"
"@
Set-Content -Path "$InstallDir\start_ipvn7.bat" -Value $startBat -Encoding ASCII

# Crear acceso directo en el Escritorio
$wshShell = New-Object -ComObject WScript.Shell
$shortcutPath = [System.IO.Path]::Combine([Environment]::GetFolderPath("Desktop"), "ipvn7 Network OS.lnk")
$shortcut = $wshShell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = "$InstallDir\start_ipvn7.bat"
$shortcut.WorkingDirectory = $InstallDir
$shortcut.Description = "Iniciar ipvn7 Network OS Dashboard"
$shortcut.Save()

# 4. Finalización y arranque
Write-Host "[4/4] Instalación completada exitosamente!" -ForegroundColor Green
Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "  Ruta de instalación: $InstallDir" -ForegroundColor White
Write-Host "  Panel Web:           http://localhost:$WebPort" -ForegroundColor White
Write-Host "  Proxy SOCKS5:        127.0.0.1:10807" -ForegroundColor White
Write-Host "  Acceso Directo:      Escritorio -> 'ipvn7 Network OS'" -ForegroundColor White
Write-Host "==========================================================" -ForegroundColor Cyan

if ($StartService) {
    Write-Host "[+] Iniciando nodo en segundo plano..." -ForegroundColor Green
    Start-Process -FilePath "$InstallDir\bin\ipvn7.exe" -ArgumentList "--port $P2PPort --web-port $WebPort" -WorkingDirectory $InstallDir
}
