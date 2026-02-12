# TakeoutFix (Русская версия)

Перенесите библиотеку из Google Photos без потери важных метаданных.
TakeoutFix обрабатывает ZIP-архивы Google Takeout и восстанавливает дату съемки, геолокацию и описание.

[English](../README.md) · [Русский](README.ru.md) · [中文](README.zh-CN.md) · [हिन्दी](README.hi.md) · [Español](README.es.md) · [Français](README.fr.md) · [العربية](README.ar.md) · [Deutsch](README.de.md)

## Почему TakeoutFix

- Сохраняет метаданные фото и видео при миграции.
- Работает со стандартными ZIP-архивами Google Takeout.
- Ведет вас по понятному пошаговому процессу в терминале.
- Подходит обычным пользователям, без скриптов и ручной рутины.

## Быстрый старт (3 шага)

1. Экспортируйте Google Photos через Google Takeout в формате ZIP.
2. Положите все `*.zip` в одну локальную папку.
3. Откройте терминал в этой папке и запустите команду ниже.

## Запуск (рекомендуется)

Запускайте прямо в папке, где лежат ZIP-архивы Takeout.

macOS/Linux:

```bash
curl -fsSL https://github.com/vchilikov/takeout-fix/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr -useb https://github.com/vchilikov/takeout-fix/releases/latest/download/install.ps1 | iex
```

## Ручной запуск (опционально)

Используйте этот вариант, если не хотите one-liner установщик.

1. Скачайте готовый бинарник для вашей ОС из [GitHub Releases](https://github.com/vchilikov/takeout-fix/releases).
2. Запустите его в папке с ZIP-архивами Takeout.

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

## Что получится на выходе

После успешной обработки:

- Готовые файлы будут в `./takeoutfix-extracted/Takeout`.
- Метаданные применятся к поддерживаемым фото и видео.
- `Tags` из JSON будут записаны в `Keywords` и `Subject`.
- Если `photoTakenTime.timestamp` в JSON отсутствует или невалиден, а имя файла начинается с `YYYY-MM-DD HH.MM.SS`, дата будет восстановлена из имени файла.
- Папку `./takeoutfix-extracted/Takeout` можно загружать в новое облако.

## Частые проблемы

- `No ZIP archives found in current folder.`
  - TakeoutFix автоматически проверяет, есть ли уже распакованный контент Takeout в рабочей папке.
  - Если сообщение осталось, положите ZIP-части Takeout в корень папки или запустите утилиту в папке с распакованным контентом.
- `Corrupt ZIP files found. Processing stopped.`
  - Скачайте поврежденные части архива заново в Google Takeout и повторите запуск.
- `Missing dependencies: exiftool`
  - Используйте рекомендованную one-liner команду выше или установите `exiftool` вручную.
- `Not enough disk space even with auto-delete enabled.`
  - Освободите место на диске и запустите повторно.
- macOS сообщает, что приложение не проверено
  - Снимите карантин и запустите снова:
    ```bash
    xattr -d com.apple.quarantine ./takeoutfix
    ```
