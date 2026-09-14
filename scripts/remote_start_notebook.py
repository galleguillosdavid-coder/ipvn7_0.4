import subprocess
import sys

ps_cmd = (
    "$arg = @{"
    "ClassName='Win32_Process'; "
    "MethodName='Create'; "
    "Arguments=@{CommandLine='cmd.exe /c C:\\ipvn7\\start_daemon.bat > C:\\ipvn7\\logs\\daemon.log 2>&1'; "
    "CurrentDirectory='C:\\ipvn7'}"
    "}; "
    "Invoke-CimMethod @arg"
)

ssh_cmd = ["ssh", "frondabrick@192.168.1.106", f'powershell -NoProfile -Command "{ps_cmd}"']
print("Ejecutando inicio remoto via WMI...")
res = subprocess.run(ssh_cmd, capture_output=True, text=True, encoding="utf-8", errors="replace")
print("Return code:", res.returncode)
print("Stdout:", res.stdout)
print("Stderr:", res.stderr)
