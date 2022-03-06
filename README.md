# TeleShell

Shell commands executor via Telegram Bot.

## Building under Windows for Linux

```ps
$Env:GOOS = "linux"; $Env:GOARCH = "amd64"
& "C:\Program Files\Go\bin\go.exe" build -o build/teleshell.linux .
```

## Running under Linux

```bash
export TELESHELL_API_TOKEN="TELEGRAM_BOT_API_TOKEN"
export TELESHELL_PASSWORD="PASSWORD_TO_ACCESS_BOT"
export TELESHELL_BASH_PATH="/bin/bash"
```