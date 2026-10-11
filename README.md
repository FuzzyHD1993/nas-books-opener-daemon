# NAS Books Opener Daemon

Локальный демон для открытия книг из `/mnt/nas/Books/` в системных приложениях.

## Что делает

- Принимает HTTP-запросы на `http://127.0.0.1:9999`
- Открывает файлы в читалке (`foliate`)
- Открывает папку в файловом менеджере (`cosmic-files`)

## API

- `GET /health` — статус
- `GET /open?name=<filename>` — открыть книгу
- `GET /open-folder` — открыть папку с книгами

## Установка

```bash
go build -o books-opener .
