$ErrorActionPreference = "SilentlyContinue"

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]"Administrator")
if (-not $isAdmin) {
    Start-Process powershell -ArgumentList "-ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs -Wait
    exit
}

Write-Host ""
Write-Host "=== Desinstallation Prank Server ===" -ForegroundColor Cyan
Write-Host ""

$dir = "$env:LOCALAPPDATA\Microsoft\PyService"

# 1. Arreter le processus
$proc = Get-Process svc -ErrorAction SilentlyContinue
if ($proc) {
    $proc | Stop-Process -Force
    Start-Sleep -Milliseconds 800
    Write-Host "[OK] Processus svc.exe arrete" -ForegroundColor Green
} else {
    Write-Host "[--] Processus svc.exe non trouve" -ForegroundColor Yellow
}

# 2. Supprimer la tache planifiee
schtasks /delete /tn "PyServiceHost" /f 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Tache planifiee PyServiceHost supprimee" -ForegroundColor Green
} else {
    Write-Host "[--] Tache planifiee non trouvee" -ForegroundColor Yellow
}

# 3. Supprimer les exclusions Defender
Remove-MpPreference -ExclusionPath "$env:TEMP" 2>$null
Write-Host "[OK] Exclusion Defender supprimee : $env:TEMP" -ForegroundColor Green

Remove-MpPreference -ExclusionPath $dir 2>$null
Write-Host "[OK] Exclusion Defender supprimee : $dir" -ForegroundColor Green

# 4. Supprimer le dossier d'installation
if (Test-Path $dir) {
    Remove-Item $dir -Recurse -Force
    Write-Host "[OK] Dossier supprime : $dir" -ForegroundColor Green
} else {
    Write-Host "[--] Dossier non trouve : $dir" -ForegroundColor Yellow
}

# 5. Nettoyer les fichiers residuels dans %TEMP%
$cleaned = 0
@("svc.exe","setup.exe") | ForEach-Object {
    $p = "$env:TEMP\$_"
    if (Test-Path $p) { Remove-Item $p -Force; $cleaned++ }
}
@("bsod*.ps1","play*.vbs","vol*.ps1","audio*.*") | ForEach-Object {
    Get-ChildItem $env:TEMP -Filter $_ | ForEach-Object { Remove-Item $_.FullName -Force; $cleaned++ }
}
if ($cleaned -gt 0) {
    Write-Host "[OK] $cleaned fichier(s) temporaire(s) supprimes" -ForegroundColor Green
} else {
    Write-Host "[--] Aucun fichier temporaire a supprimer" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Desinstallation terminee ===" -ForegroundColor Cyan
Write-Host ""
Read-Host "Appuie sur Entree pour fermer"
