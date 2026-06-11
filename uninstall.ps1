$ErrorActionPreference = "SilentlyContinue"

$dir = "$env:LOCALAPPDATA\Microsoft\PyService"

# Arret du processus
Get-Process -Name "svc" | Where-Object {
    $_.MainModule.FileName -like "*PyService*"
} | Stop-Process -Force

# Suppression de la tache planifiee
schtasks /delete /tn "PyServiceHost" /f 2>$null

# Suppression des fichiers
if (Test-Path $dir) {
    Remove-Item $dir -Recurse -Force
}
