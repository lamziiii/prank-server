$ErrorActionPreference = "SilentlyContinue"

# ── Auto-elevation UAC ────────────────────────────────────────────────────────
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]"Administrator")
if (-not $isAdmin) {
    Start-Process powershell -ArgumentList "-ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs -Wait
    exit
}

Write-Host ""
Write-Host "=== Desinstallation Prank Server ===" -ForegroundColor Cyan
Write-Host ""

# ── Arret du processus ────────────────────────────────────────────────────────
$dir = "$env:LOCALAPPDATA\Microsoft\PyService"
$proc = Get-Process svc -ErrorAction SilentlyContinue | Where-Object { $_.MainModule.FileName -like "*PyService*" }
if ($proc) {
    $proc | Stop-Process -Force
    Write-Host "[OK] Processus svc.exe arrete" -ForegroundColor Green
} else {
    Write-Host "[--] Processus svc.exe non trouve" -ForegroundColor Yellow
}

# ── Suppression de la tache planifiee ────────────────────────────────────────
schtasks /query /tn "PyServiceHost" 2>$null | Out-Null
if ($?) {
    schtasks /delete /tn "PyServiceHost" /f 2>$null
    Write-Host "[OK] Tache planifiee PyServiceHost supprimee" -ForegroundColor Green
} else {
    Write-Host "[--] Tache planifiee non trouvee" -ForegroundColor Yellow
}

# ── Suppression des exclusions Defender ──────────────────────────────────────
Remove-MpPreference -ExclusionPath "$env:TEMP" 2>$null
Write-Host "[OK] Exclusion Defender supprimee : $env:TEMP" -ForegroundColor Green

Remove-MpPreference -ExclusionPath $dir 2>$null
Write-Host "[OK] Exclusion Defender supprimee : $dir" -ForegroundColor Green

# ── Suppression du dossier d'installation ────────────────────────────────────
if (Test-Path $dir) {
    Remove-Item $dir -Recurse -Force
    Write-Host "[OK] Dossier supprime : $dir" -ForegroundColor Green
} else {
    Write-Host "[--] Dossier non trouve : $dir" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Desinstallation terminee ===" -ForegroundColor Cyan
Write-Host ""
Read-Host "Appuie sur Entree pour fermer"
