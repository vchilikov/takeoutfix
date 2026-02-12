# TakeoutFix (Español)

Migra tu biblioteca de Google Photos sin perder metadatos importantes.
TakeoutFix procesa archivos ZIP de Google Takeout y restaura campos como fecha de captura, ubicación y descripción.

[English](../README.md) · [Русский](README.ru.md) · [中文](README.zh-CN.md) · [हिन्दी](README.hi.md) · [Español](README.es.md) · [Français](README.fr.md) · [العربية](README.ar.md) · [Deutsch](README.de.md)

## Por qué TakeoutFix

- Conserva metadatos de fotos y videos que suelen perderse en una migración.
- Funciona directamente con los ZIP estándar de Google Takeout.
- Te guía con un flujo claro en la terminal, de principio a fin.
- Está pensado para usuarios normales, sin necesidad de scripts.

## Inicio rápido (3 pasos)

1. Exporta tu archivo de Google Photos desde Google Takeout en formato ZIP.
2. Coloca todos los `*.zip` en una sola carpeta local.
3. Abre una terminal en esa carpeta y ejecuta el comando recomendado.

## Ejecutar (Recomendado)

Ejecuta el comando directamente en la carpeta que contiene tus ZIP de Takeout.

macOS/Linux:

```bash
curl -fsSL https://github.com/vchilikov/takeout-fix/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr -useb https://github.com/vchilikov/takeout-fix/releases/latest/download/install.ps1 | iex
```

## Ejecución manual (Opcional)

Usa esta opción solo si no quieres usar el instalador one-liner.

1. Descarga el binario más reciente para tu sistema desde [GitHub Releases](https://github.com/vchilikov/takeout-fix/releases).
2. Ejecútalo en la carpeta que contiene tus ZIP de Takeout.

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

## Qué obtienes

Después de una ejecución correcta:

- Tus archivos procesados estarán en `./takeoutfix-extracted/Takeout`.
- Se aplicarán metadatos a fotos y videos compatibles.
- Los `Tags` del JSON se escribirán en `Keywords` y `Subject`.
- Si el timestamp de captura en JSON falta o es inválido, y el nombre del archivo empieza con `YYYY-MM-DD HH.MM.SS`, la fecha se restaurará desde el nombre.
- Puedes subir `./takeoutfix-extracted/Takeout` a tu nuevo almacenamiento.

## Problemas comunes

- `No ZIP archives found in current folder.`
  - TakeoutFix detecta automáticamente contenido de Takeout ya extraído en la carpeta de trabajo.
  - Si el mensaje continúa, coloca todas las partes ZIP de Takeout en el nivel superior de la carpeta o ejecuta la herramienta desde una carpeta con contenido de Takeout ya extraído.
- `Corrupt ZIP files found. Processing stopped.`
  - Vuelve a descargar las partes dañadas desde Google Takeout y reintenta.
- `Missing dependencies: exiftool`
  - Usa el comando one-liner recomendado arriba, o instala `exiftool` manualmente.
- `Not enough disk space even with auto-delete enabled.`
  - Libera espacio en disco y vuelve a ejecutar.
- macOS indica que la app no está verificada
  - Quita la cuarentena y vuelve a ejecutar:
    ```bash
    xattr -d com.apple.quarantine ./takeoutfix
    ```
