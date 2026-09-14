# Script de Sincronización Automática al Notebook
param(
    [string]$NotebookIP = "192.168.1.106",
    [string]$NotebookUser = "frondabrick",
    [int]$P2PPort = 7778,
    [int]$WebPort = 8080
)

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   IPv7 v0.4: Sincronizador Autónomo a Notebook ($NotebookIP)   " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Esperar a que el notebook esté accesible en LAN si está desconectado
Write-Host "[*] Verificando presencia de $NotebookIP en la red..." -ForegroundColor Yellow
$connected = $false
for ($i = 1; $i -le 30; $i++) {
    if (Test-Connection -ComputerName $NotebookIP -Count 1 -Quiet) {
        $connected = $true
        break
    }
    Write-Host "    [Intento $i/30] Esperando conexión de $NotebookIP... (conecta el notebook a la LAN por 10s si aún no lo has hecho)" -ForegroundColor Gray
    Start-Sleep -Seconds 2
}

if (-not $connected) {
    Write-Host "[-] El notebook no responde en $NotebookIP. Si cambió de IP en la LAN, pásale la IP como parámetro: .\scripts\sync_to_notebook.ps1 -NotebookIP <IP>" -ForegroundColor Red
    exit 1
}

Write-Host "[+] Notebook detectado en $NotebookIP. Iniciando transferencia de la versión 0.4.0..." -ForegroundColor Green

# 2. Detener proceso previo en notebook si existe
Write-Host "[1/4] Deteniendo proceso previo de ipvn7 en el notebook..." -ForegroundColor Yellow
ssh -o StrictHostKeyChecking=no "$NotebookUser@$NotebookIP" "taskkill /F /IM ipvn7.exe" 2>$null

# 3. Transferir binario Windows amd64 y assets web
Write-Host "[2/4] Copiando binarios compilados v0.4.0 (EBRA + STUN + UI Nivel 0)..." -ForegroundColor Yellow
scp -o StrictHostKeyChecking=no .\bin\windows_amd64\ipvn7.exe "$NotebookUser@${NotebookIP}:C:/ipvn7/bin/ipvn7.exe"
scp -o StrictHostKeyChecking=no .\bin\windows_amd64\ipvn7.exe "$NotebookUser@${NotebookIP}:C:/ipvn7/ipvn7.exe"
scp -o StrictHostKeyChecking=no .\bin\windows_amd64\ipvn7-cli.exe "$NotebookUser@${NotebookIP}:C:/ipvn7/bin/ipvn7-cli.exe"
scp -o StrictHostKeyChecking=no -r .\web\* "$NotebookUser@${NotebookIP}:C:/ipvn7/web/"

# 4. Actualizar start_daemon.bat
Write-Host "[3/4] Configurando script de inicio en el Notebook..." -ForegroundColor Yellow
$batContent = "C:\ipvn7\bin\ipvn7.exe -port $P2PPort -web-port $WebPort"
ssh -o StrictHostKeyChecking=no "$NotebookUser@$NotebookIP" "echo $batContent > C:\ipvn7\start_daemon.bat"

# 5. Iniciar servicio remoto mediante WMI
Write-Host "[4/4] Levantando ipvn7 v0.4.0 en el Notebook mediante WMI..." -ForegroundColor Yellow
python scripts/remote_start_notebook.py

Start-Sleep -Seconds 3

Write-Host ""
Write-Host "==========================================================" -ForegroundColor Green
Write-Host "   ¡DESPLIEGUE v0.4.0 COMPLETADO EN EL NOTEBOOK!           " -ForegroundColor Green
Write-Host "==========================================================" -ForegroundColor Green
Write-Host "1. El nodo ya está ejecutándose en el notebook." -ForegroundColor White
Write-Host "2. AHORA PUEDES DESCONECTAR EL NOTEBOOK DE LA RED LAN" -ForegroundColor Cyan
Write-Host "   (conéctalo al hotspot 4G del teléfono o a otra red)." -ForegroundColor Cyan
Write-Host "3. Los motores EBRA y STUN de ambos nodos se descubrirán solos." -ForegroundColor Yellow
