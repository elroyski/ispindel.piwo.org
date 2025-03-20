#!/bin/bash

# Konfiguracja
APP_NAME="ispindel"
APP_DIR="/home/piwo/domains/ispindle.piwo.org/ispindle-web"
PORT="49330"

# Sprawdzenie czy jesteśmy w odpowiednim katalogu
CURRENT_DIR=$(pwd)
if [ "$CURRENT_DIR" != "$APP_DIR" ]; then
    echo "Błąd: Skrypt musi być uruchomiony w katalogu $APP_DIR"
    echo "Aktualny katalog: $CURRENT_DIR"
    echo "Przechodzę do właściwego katalogu..."
    cd "$APP_DIR"
fi

# Sprawdzenie czy plik aplikacji istnieje
if [ ! -f "$APP_NAME" ]; then
    echo "Błąd: Nie znaleziono pliku aplikacji $APP_NAME!"
    echo "Upewnij się, że aplikacja została skompilowana używając setup_server.sh"
    exit 1
fi

# Zatrzymanie poprzedniej instancji
echo "Zatrzymywanie poprzedniej instancji..."
pkill -f $APP_NAME

# Uruchomienie nowej instancji
echo "Uruchamianie aplikacji..."
nohup ./$APP_NAME > app.log 2>&1 &

# Sprawdzenie czy aplikacja się uruchomiła
sleep 2
if pgrep -f $APP_NAME > /dev/null; then
    echo "Aplikacja została pomyślnie uruchomiona!"
    echo "Logi dostępne w pliku app.log"
else
    echo "Wystąpił problem z uruchomieniem aplikacji."
    echo "Sprawdź logi w pliku app.log"
fi 