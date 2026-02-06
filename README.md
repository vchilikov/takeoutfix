# TakeoutFix

TakeoutFix помогает аккуратно восстановить метаданные Google Photos Takeout и записать их обратно в фото/видео.

Проект решает типовые проблемы экспорта:

- берет дату, GPS и описание из `.json` рядом с медиа;
- исправляет неверные расширения файлов;
- добавляет `OffsetTime*` (timezone offset), когда его можно вычислить по координатам;
- поддерживает повторный запуск (идемпотентное поведение для `fix`);
- удаляет только использованные `.json` в режиме `clean-json`.

## Что нужно перед запуском

- [Go](https://go.dev/) 1.22+
- [ExifTool](https://exiftool.org) в `PATH`

Проверка:

```bash
exiftool -ver
go version
```

## Быстрый старт

1. Собери бинарник:

```bash
go build -o takeoutfix
```

2. Запусти исправление метаданных:

```bash
takeoutfix fix ~/Downloads/Takeout/Google\ Photos
```

3. Опционально удали использованные JSON:

```bash
takeoutfix clean-json ~/Downloads/Takeout/Google\ Photos
```

Если путь содержит пробелы, используй `\` или кавычки.

## Как устроены входные данные

Ожидается стандартная структура Takeout:

```text
Takeout/
  Google Photos/
    Album 1/
      IMG_0001.jpg
      IMG_0001.jpg.json
      IMG_0002.mp4
      IMG_0002.mp4.json
```

## Команды

### `fix`

Сопоставляет медиа и JSON, исправляет расширения, пишет метаданные через ExifTool.

```bash
takeoutfix fix <path-to-google-photos-folder>
```

Пример с сохранением лога:

```bash
takeoutfix fix <path> | tee takeoutfix.log
```

### `clean-json`

Удаляет только те `.json`, которые были успешно сопоставлены с медиа.  
Неиспользованные/осиротевшие `.json` остаются и логируются.

```bash
takeoutfix clean-json <path-to-google-photos-folder>
```

## Поведение и ограничения

- Если timezone offset нельзя вычислить (нет GPS или lookup неуспешен), обработка продолжается без `OffsetTime*`.
- Для форматов без прямой поддержки ExifTool метаданные пишутся в `.xmp` sidecar.
- Проект ориентирован на структуру Google Photos Takeout.

## Локальная проверка

```bash
go test ./...
go vet ./...
go build ./...
```

## Релизы

Релизы автоматизированы через GitHub Actions + GoReleaser.

Публикация версии:

```bash
git tag vX.Y.Z
git push --tags
```

После пуша тега создается GitHub Release с бинарниками для Linux/macOS/Windows (`amd64` + `arm64`) и файлом `checksums.txt`.

Скачать релизы: [https://github.com/vchilikov/takeout-fix/releases](https://github.com/vchilikov/takeout-fix/releases)
