# TakeoutFix (Deutsch)

Migriere deine Google-Photos-Bibliothek, ohne wichtige Metadaten zu verlieren.
TakeoutFix verarbeitet Google-Takeout-ZIP-Dateien und stellt Felder wie Aufnahmedatum, Standort und Beschreibung wieder her.

[English](../README.md) · [Русский](README.ru.md) · [中文](README.zh-CN.md) · [हिन्दी](README.hi.md) · [Español](README.es.md) · [Français](README.fr.md) · [العربية](README.ar.md) · [Deutsch](README.de.md)

## Warum TakeoutFix

- Bewahrt Foto- und Video-Metadaten, die bei Migrationen oft verloren gehen.
- Arbeitet direkt mit den standardmaessigen Google-Takeout-ZIP-Exporten.
- Fuehrt dich mit einem klaren Terminal-Ablauf Schritt fuer Schritt durch den Prozess.
- Fuer normale Nutzer gedacht, ohne Skripting.

## Schnellstart (3 Schritte)

1. Exportiere dein Google-Photos-Archiv ueber Google Takeout als ZIP-Dateien.
2. Lege alle `*.zip`-Dateien in einen lokalen Ordner.
3. Oeffne ein Terminal in diesem Ordner und fuehre den empfohlenen Befehl aus.

## Ausfuehren (Empfohlen)

Fuehre den Befehl direkt in dem Ordner mit deinen Takeout-ZIP-Dateien aus.

macOS/Linux:

```bash
curl -fsSL https://github.com/vchilikov/takeout-fix/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr -useb https://github.com/vchilikov/takeout-fix/releases/latest/download/install.ps1 | iex
```

## Manuelle Ausfuehrung (Optional)

Nutze diese Option nur, wenn du den One-Liner-Installer nicht verwenden willst.

1. Lade die neueste Binary fuer dein Betriebssystem von [GitHub Releases](https://github.com/vchilikov/takeout-fix/releases) herunter.
2. Starte sie im Ordner mit deinen Takeout-ZIP-Dateien.

macOS/Linux:

```bash
./takeoutfix
./takeoutfix --workdir /path/to/folder
```

Windows (PowerShell):

```powershell
.\takeoutfix.exe
.\takeoutfix.exe --workdir C:\path\to\folder
```

## Ergebnis

Nach erfolgreichem Lauf:

- Deine verarbeiteten Dateien liegen in `./takeoutfix-extracted/Takeout`.
- Metadaten werden auf unterstuetzte Fotos und Videos angewendet.
- JSON-`Tags` werden in `Keywords` und `Subject` geschrieben.
- Wenn der Aufnahme-Zeitstempel im JSON fehlt oder ungueltig ist und der Dateiname mit `YYYY-MM-DD HH.MM.SS` beginnt, wird das Datum aus dem Dateinamen wiederhergestellt.
- Ein detaillierter Laufbericht wird unter `./.takeoutfix/reports/report-YYYYMMDD-HHMMSS.json` gespeichert.
- Du kannst `./takeoutfix-extracted/Takeout` in deinen neuen Speicher hochladen.

## Haeufige Probleme

- `No ZIP files or extracted Takeout data found in this folder.`
  - Lege alle Takeout-ZIP-Teile in die oberste Ebene des Arbeitsordners oder starte im Ordner mit bereits entpacktem Takeout-Inhalt.
- `Some ZIP files are corrupted. Please re-download them and run again.`
  - Lade beschaedigte Archivteile aus Google Takeout neu herunter und starte erneut.
- `Step 1/3: Checking dependencies... missing`
  - Nutze den empfohlenen One-Liner oben oder installiere `exiftool` manuell.
- `Not enough free disk space to continue.`
  - Schaffe mehr freien Speicherplatz und starte erneut.
- macOS meldet, dass die App nicht verifiziert ist
  - Entferne das Quarantaene-Attribut und starte erneut:
    ```bash
    xattr -d com.apple.quarantine ./takeoutfix
    ```
