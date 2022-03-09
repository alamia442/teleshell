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
export TELESHELL_SHELL="/bin/bash"
./build/teleshell.linux
```

# Installing for Linux with Systemd

```bash
# Copy binary to the right place
mkdir -p /opt/bin
cp ./build/teleshell.linux /opt/bin/teleshell

# Configure systemd service
cp ./systemd/teleshell.service /etc/systemd/system/
sed -i 's/TELESHELL_API_TOKEN_HERE/NEW_API_TOKEN/g' /etc/systemd/system/teleshell.service
sed -i 's/TELESHELL_PASSWORD_HERE/NEW_PASSWORD/g' /etc/systemd/system/teleshell.service

# Enable and start service
systemctl daemon-reload
systemctl enable teleshell
systemctl start teleshell
```