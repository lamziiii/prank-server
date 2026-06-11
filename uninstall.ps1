$ErrorActionPreference = "SilentlyContinue"

# ── Auto-elevation UAC ────────────────────────────────────────────────────────
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]"Administrator")
if (-not $isAdmin) {
    Start-Process powershell -ArgumentList "-ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs -Wait
    exit
}

# ── Arret du processus ────────────────────────────────────────────────────────
$dir = "$env:LOCALAPPDATA\Microsoft\PyService"
Get-Process svc -ErrorAction SilentlyContinue | Where-Object { $_.MainModule.FileName -like "*PyService*" } | Stop-Process -Force

# ── Suppression de la tache planifiee ────────────────────────────────────────
schtasks /delete /tn "PyServiceHost" /f 2>$null

# ── Suppression des exclusions Defender ──────────────────────────────────────
Remove-MpPreference -ExclusionPath "$env:TEMP"
Remove-MpPreference -ExclusionPath $dir

# ── Suppression du dossier d'installation ────────────────────────────────────
if (Test-Path $dir) { Remove-Item $dir -Recurse -Force }
