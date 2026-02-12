# TakeoutFix (Français)

Migrez votre bibliothèque Google Photos sans perdre de métadonnées importantes.
TakeoutFix traite les archives ZIP Google Takeout et restaure des champs comme la date de prise de vue, la localisation et la description.

[English](../README.md) · [Русский](README.ru.md) · [中文](README.zh-CN.md) · [हिन्दी](README.hi.md) · [Español](README.es.md) · [Français](README.fr.md) · [العربية](README.ar.md) · [Deutsch](README.de.md)

## Pourquoi TakeoutFix

- Préserve les métadonnées photo/vidéo souvent perdues pendant une migration.
- Fonctionne directement avec les exports ZIP standards de Google Takeout.
- Propose un flux terminal clair, guidé de bout en bout.
- Conçu pour les utilisateurs non techniques, sans scripts.

## Démarrage rapide (3 étapes)

1. Exportez vos Google Photos depuis Google Takeout au format ZIP.
2. Placez tous les fichiers `*.zip` dans un seul dossier local.
3. Ouvrez un terminal dans ce dossier et lancez la commande recommandée.

## Exécution (Recommandée)

Lancez la commande directement dans le dossier contenant vos ZIP Takeout.

macOS/Linux:

```bash
curl -fsSL https://github.com/vchilikov/takeout-fix/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr -useb https://github.com/vchilikov/takeout-fix/releases/latest/download/install.ps1 | iex
```

## Exécution manuelle (Optionnelle)

Utilisez cette option uniquement si vous ne voulez pas utiliser l'installeur one-liner.

1. Téléchargez le binaire le plus récent pour votre OS depuis [GitHub Releases](https://github.com/vchilikov/takeout-fix/releases).
2. Exécutez-le dans le dossier contenant vos ZIP Takeout.

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

## Résultat obtenu

Après une exécution réussie :

- Vos médias traités se trouvent dans `./takeoutfix-extracted/Takeout`.
- Les métadonnées sont appliquées aux photos et vidéos prises en charge.
- Les `Tags` JSON sont écrits dans `Keywords` et `Subject`.
- Si `photoTakenTime.timestamp` est absent ou invalide et que le nom de fichier commence par `YYYY-MM-DD HH.MM.SS`, la date est restaurée depuis le nom de fichier.
- Vous pouvez importer `./takeoutfix-extracted/Takeout` dans votre nouveau stockage.

## Problèmes fréquents

- `No ZIP archives found in current folder.`
  - TakeoutFix détecte automatiquement le contenu Takeout déjà extrait dans le dossier de travail.
  - Si le message persiste, placez toutes les parties ZIP Takeout à la racine du dossier de travail, ou lancez l'outil depuis un dossier contenant déjà le contenu Takeout extrait.
- `Corrupt ZIP files found. Processing stopped.`
  - Retéléchargez les parties corrompues depuis Google Takeout puis relancez.
- `Missing dependencies: exiftool`
  - Utilisez la commande one-liner recommandée ci-dessus, ou installez `exiftool` manuellement.
- `Not enough disk space even with auto-delete enabled.`
  - Libérez de l'espace disque puis relancez.
- macOS indique que l'application n'est pas vérifiée
  - Retirez la quarantaine puis relancez :
    ```bash
    xattr -d com.apple.quarantine ./takeoutfix
    ```
