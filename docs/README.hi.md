# TakeoutFix (हिंदी)

महत्वपूर्ण मेटाडेटा खोए बिना अपनी Google Photos लाइब्रेरी माइग्रेट करें।
TakeoutFix Google Takeout ZIP फाइलों को प्रोसेस करता है और capture date, location, और description जैसी जानकारी वापस लिखता है।

[English](../README.md) · [Русский](README.ru.md) · [中文](README.zh-CN.md) · [हिन्दी](README.hi.md) · [Español](README.es.md) · [Français](README.fr.md) · [العربية](README.ar.md) · [Deutsch](README.de.md)

## TakeoutFix क्यों

- माइग्रेशन के दौरान फोटो/वीडियो का जरूरी मेटाडेटा बचाने में मदद करता है।
- Google Takeout के standard ZIP export के साथ सीधे काम करता है।
- शुरू से अंत तक साफ, guided terminal flow देता है।
- आम users के लिए बना है, scripting की जरूरत नहीं।

## Quick Start (3 कदम)

1. Google Takeout से अपनी Google Photos archive को ZIP के रूप में export करें।
2. सभी `*.zip` फाइलों को एक local folder में रखें।
3. उसी folder में terminal खोलें और नीचे दिया recommended command चलाएं।

## Run (Recommended)

कमांड उसी folder में चलाएं जिसमें आपकी Takeout ZIP फाइलें हैं।

macOS/Linux:

```bash
curl -fsSL https://github.com/vchilikov/takeout-fix/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr -useb https://github.com/vchilikov/takeout-fix/releases/latest/download/install.ps1 | iex
```

## Manual Run (Optional)

अगर आप one-liner installer नहीं इस्तेमाल करना चाहते, तो यह तरीका अपनाएं।

1. अपने OS के लिए latest binary [GitHub Releases](https://github.com/vchilikov/takeout-fix/releases) से डाउनलोड करें।
2. इसे उस folder में चलाएं जिसमें Takeout ZIP फाइलें हों।

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

## आपको क्या मिलेगा

सफल run के बाद:

- Processed files `./takeoutfix-extracted/Takeout` में मिलेंगी।
- Supported photos और videos में metadata apply हो जाएगा।
- आप `./takeoutfix-extracted/Takeout` को अपने नए storage में upload कर सकते हैं।

## Common Issues

- `No ZIP archives found in current folder.`
  - सभी Takeout ZIP parts को working folder के top level में रखकर फिर चलाएं।
- `Corrupt ZIP files found. Processing stopped.`
  - Google Takeout से खराब archive parts दोबारा डाउनलोड करें और rerun करें।
- `Missing dependencies: exiftool`
  - ऊपर दिया recommended one-liner command चलाएं, या `exiftool` manually install करें।
- `Not enough disk space even with auto-delete enabled.`
  - डिस्क में space खाली करें और दोबारा चलाएं।
- macOS app verify error दिखाता है
  - quarantine हटाकर फिर चलाएं:
    ```bash
    xattr -d com.apple.quarantine ./takeoutfix
    ```
