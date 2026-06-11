$ErrorActionPreference = "SilentlyContinue"

$dir = "$env:LOCALAPPDATA\Microsoft\PyService"

Get-Process svc | Where-Object { $_.MainModule.FileName -like "*PyService*" } | Stop-Process -Force

schtasks /delete /tn "PyServiceHost" /f 2>$null

if (Test-Path $dir) { Remove-Item $dir -Recurse -Force }
