# Script de despliegue y conexión persistente con el Notebook (192.168.1.106)
param(
    [string]$NotebookIP = "192.168.1.106",
    [string]$NotebookUser = "Dvd",
    [int]$P2PPort = 7001,
    [int]$WebPort = 8080
)

$ErrorActionPreference = "Continue"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   IPv7 v0.4: Despliegue Persistente a Notebook ($NotebookIP)   " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Probar SSH sin contraseña
Write-Host "[1/5] Verificando handshake SSH con $NotebookUser@$NotebookIP..." -ForegroundColor Yellow
$sshTest = ssh -o BatchMode=yes -o ConnectTimeout=5 -o StrictHostKeyChecking=no "$NotebookUser@$NotebookIP" "echo OK" 2>&1
if ($sshTest -notmatch "OK") {
    Write-Host "[-] SSH no autenticado aún. Error: $sshTest" -ForegroundColor Red
    Write-Host "[!] Asegúrate de haber ejecutado el comando de autorización en el PowerShell del notebook." -ForegroundColor Magenta
    exit 1
}
Write-Host "[+] Autenticación SSH persistente establecida exitosamente." -ForegroundColor Green

# 2. Crear carpetas de destino en el Notebook
Write-Host "[2/5] Preparando directorios en el Notebook..." -ForegroundColor Yellow
ssh -o StrictHostKeyChecking=no "$NotebookUser@$NotebookIP" "powershell -Command New-Item -ItemType Directory -Force -Path 'C:\ipvn7\bin', 'C:\ipvn7\web'" | Out-Null
ssh -o StrictHostKeyChecking=no "$NotebookUser@$NotebookIP" "wsl -d Ubuntu -e mkdir -p /home/$NotebookUser/ipvn7/web" 2>$null

# 3. Transferir binarios (Linux para WSL2 y Windows nativo)
Write-Host "[3/5] Transfiriendo binarios y assets compilados..." -ForegroundColor Yellow
scp -o StrictHostKeyChecking=no .\bin\linux_amd64\ipvn7 "$NotebookUser@${NotebookIP}:C:/ipvn7/bin/ipvn7"
scp -o StrictHostKeyChecking=no .\bin\linux_amd64\ipvn7-cli "$NotebookUser@${NotebookIP}:C:/ipvn7/bin/ipvn7-cli"
scp -o StrictHostKeyChecking=no .\bin\windows_amd64\ipvn7.exe "$NotebookUser@${NotebookIP}:C:/ipvn7/bin/ipvn7.exe"
scp -o StrictHostKeyChecking=no -r .\web\* "$NotebookUser@${NotebookIP}:C:/ipvn7/web/"

# 4. Mover binario a WSL2 en el notebook y otorgar permisos
Write-Host "[4/5] Instalando binario en WSL2 del Notebook..." -ForegroundColor Yellow
ssh -o StrictHostKeyChecking=no "$NotebookUser@$NotebookIP" "wsl -d Ubuntu -e bash -c 'cp /mnt/c/ipvn7/bin/ipvn7 /home/$NotebookUser/ipvn7/ && cp /mnt/c/ipvn7/bin/ipvn7-cli /home/$NotebookUser/ipvn7/ && cp -r /mnt/c/ipvn7/web /home/$NotebookUser/ipvn7/ && chmod +x /home/$NotebookUser/ipvn7/ipvn7 /home/$NotebookUser/ipvn7/ipvn7-cli'"

# 5. Iniciar servicio persistente en WSL2 del Notebook
Write-Host "[5/5] Iniciando demonio IPv7 en WSL2 del Notebook (P2P :$P2PPort, Web :$WebPort)..." -ForegroundColor Yellow
$launchCmd = "nohup /home/$NotebookUser/ipvn7/ipvn7 --port $P2PPort --web-port $WebPort > /home/$NotebookUser/ipvn7/ipvn7.log 2>&1 &"
ssh -o StrictHostKeyChecking=no "$NotebookUser@$NotebookIP" "wsl -d Ubuntu -e bash -c '$launchCmd'"

Start-Sleep -Seconds 2

# Verificar ejecución remota
$checkProc = ssh -o StrictHostKeyChecking=no "$NotebookUser@$NotebookIP" "wsl -d Ubuntu -e ps aux" 2>&1
if ($checkProc -match "ipvn7") {
    Write-Host "[+] ¡Nodo IPv7 v0.4 corriendo exitosamente en el Notebook!" -ForegroundColor Green
    Write-Host "    Web UI Notebook: http://${NotebookIP}:${WebPort}" -ForegroundColor Cyan
    Write-Host "    P2P UDP Listen: ${NotebookIP}:${P2PPort}" -ForegroundColor Cyan
} else {
    Write-Host "[!] El proceso se lanzó, revisando logs:" -ForegroundColor Yellow
    ssh -o StrictHostKeyChecking=no "$NotebookUser@$NotebookIP" "wsl -d Ubuntu -e cat /home/$NotebookUser/ipvn7/ipvn7.log"
}
