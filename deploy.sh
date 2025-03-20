#!/bin/bash

# Konfiguracja
DEVIL_USER="twoj_username"  # Zmień na swojego użytkownika na mydevil.net
DEVIL_HOST="twoj_username.mydevil.net"  # Zmień na swoją domenę
APP_NAME="ispindel"
REMOTE_DIR="/home/$DEVIL_USER/$APP_NAME"

# Budowanie aplikacji
echo "Budowanie aplikacji..."
GOOS=linux GOARCH=amd64 go build -o $APP_NAME cmd/server/main.go

# Tworzenie archiwum
echo "Tworzenie archiwum..."
tar -czf $APP_NAME.tar.gz $APP_NAME web/templates/*

# Kopiowanie na serwer
echo "Kopiowanie na serwer..."
scp $APP_NAME.tar.gz $DEVIL_USER@$DEVIL_HOST:$REMOTE_DIR/

# Wykonanie komend na serwerze
echo "Konfiguracja na serwerze..."
ssh $DEVIL_USER@$DEVIL_HOST "cd $REMOTE_DIR && \
    tar -xzf $APP_NAME.tar.gz && \
    rm $APP_NAME.tar.gz && \
    chmod +x $APP_NAME && \
    ./restart.sh"

# Czyszczenie lokalne
rm $APP_NAME.tar.gz $APP_NAME

echo "Deployment zakończony!" 