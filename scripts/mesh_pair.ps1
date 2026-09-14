# Script de emparejamiento y ejecución de tareas distribuidas entre PC y Notebook
param(
    [string]$LocalIP = "192.168.1.198",
    [int]$LocalP2PPort = 7777,
    [string]$NotebookIP = "192.168.1.106",
    [int]$NotebookP2PPort = 7001
)

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   IPv7 v0.4: Emparejamiento Mesh y Tareas Distribuidas    " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Obtener status y DID local
Write-Host "[1/4] Consultando estado del nodo local en WSL2..." -ForegroundColor Yellow
$localStatus = wsl -d Ubuntu -e curl -s http://localhost:7070/api/status | ConvertFrom-Json
Write-Host "  -> Local Node DID: $($localStatus.did)" -ForegroundColor Green

# 2. Obtener status y DID del notebook
Write-Host "[2/4] Consultando estado del nodo Notebook..." -ForegroundColor Yellow
$notebookStatus = curl.exe -s "http://${NotebookIP}:8080/api/status" | ConvertFrom-Json
Write-Host "  -> Notebook Node DID: $($notebookStatus.did)" -ForegroundColor Green

# 3. Emparejar Notebook con Nodo Local
Write-Host "[3/4] Enviando paquete de Handshake Noise XX & Registro en Kùzu..." -ForegroundColor Yellow
# Conectar localmente hacia el notebook
$pairReq = @{
    peer_address = "${NotebookIP}:${NotebookP2PPort}"
} | ConvertTo-Json
$pairResult = wsl -d Ubuntu -e curl -s -X POST -H "Content-Type: application/json" -d $pairReq http://localhost:7070/api/peers/connect

Write-Host "  -> Resultado de conexión local: $pairResult" -ForegroundColor Cyan

# 4. Monitorear Telemetría y Anillo Kleinberg
Write-Host "[4/4] Verificando estado del radar y métricas de capa L1/L2..." -ForegroundColor Yellow
$telemetry = wsl -d Ubuntu -e curl -s http://localhost:7070/api/telemetry
Write-Host "  -> Telemetría L1/L2 activa:" -ForegroundColor Green
Write-Host $telemetry

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "Panel Visual Kleinberg disponible en: http://localhost:7070" -ForegroundColor Green
Write-Host "Panel Visual Notebook disponible en: http://${NotebookIP}:8080" -ForegroundColor Green
Write-Host "==========================================================" -ForegroundColor Cyan
