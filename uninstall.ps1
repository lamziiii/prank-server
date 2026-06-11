$ErrorActionPreference = "SilentlyContinue"

$dir = "$env:LOCALAPPDATA\Microsoft\PyService"

Get-Process -Name pythonw, python | Where-Object {
    (Get-WmiObject Win32_Process -Filter "ProcessId=$($_.Id)").CommandLine -like "*PyService*"
} | Stop-Process -Force

schtasks /delete /tn "PyServiceHost" /f 2>$null

if (Test-Path $dir) { Remove-Item $dir -Recurse -Force }
