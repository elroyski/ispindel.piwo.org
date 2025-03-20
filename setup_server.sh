#!/bin/bash

# Konfiguracja
APP_NAME="ispindel"
BASE_DIR="/home/piwo/domains/ispindle.piwo.org"
APP_DIR="$BASE_DIR/ispindle-web"
GIT_REPO="https://github.com/elroyski/ispindel.piwo.org.git"

# Sprawdzenie czy jesteśmy w odpowiednim katalogu
CURRENT_DIR=$(pwd)
if [ "$CURRENT_DIR" != "$BASE_DIR" ]; then
    echo "Błąd: Skrypt musi być uruchomiony w katalogu $BASE_DIR"
    echo "Aktualny katalog: $CURRENT_DIR"
    exit 1
fi

# Sprawdzenie uprawnień
if [ ! -w "$BASE_DIR" ]; then
    echo "Błąd: Brak uprawnień do zapisu w katalogu $BASE_DIR"
    exit 1
fi

# Usunięcie starego katalogu jeśli istnieje
if [ -d "$APP_DIR" ]; then
    echo "Usuwanie starego katalogu $APP_DIR..."
    rm -rf "$APP_DIR"
fi

# Klonowanie repozytorium do nowego katalogu
echo "Klonowanie repozytorium..."
if ! git clone $GIT_REPO "$APP_DIR"; then
    echo "Błąd: Nie udało się sklonować repozytorium!"
    echo "Sprawdź czy masz dostęp do repozytorium i czy ścieżka jest poprawna."
    exit 1
fi

# Przejście do katalogu aplikacji
cd "$APP_DIR"

# Sprawdzenie czy struktura katalogów jest poprawna
if [ ! -d "cmd/server" ]; then
    echo "Błąd: Nie znaleziono katalogu cmd/server!"
    echo "Struktura katalogów:"
    ls -R
    exit 1
fi

# Tworzenie pliku .env
echo "Tworzenie pliku konfiguracyjnego..."
cat > .env << EOL
DB_HOST=pgsql18.mydevil.net
DB_USER=p1270_ispindle
DB_PASSWORD=Kochanapysia1
DB_NAME=p1270_ispindle
DB_PORT=5432
JWT_SECRET=twoj-tajny-klucz-jwt
PORT=49330
EOL

# Kompilacja aplikacji
echo "Kompilacja aplikacji..."
echo "Sprawdzanie wersji Go..."
go version
echo "Inicjalizacja modułu Go..."
go mod init ispindel.piwo.org
echo "Pobieranie zależności..."
go mod tidy
echo "Kompilacja aplikacji..."
go build -v -o $APP_NAME ./cmd/server

# Sprawdzenie czy plik został utworzony
if [ -f "$APP_NAME" ]; then
    echo "Aplikacja została pomyślnie skompilowana!"
    ls -l $APP_NAME
else
    echo "Błąd: Nie udało się skompilować aplikacji!"
    exit 1
fi

# Nadanie uprawnień wykonywania
chmod +x $APP_NAME
chmod +x restart.sh

echo "Instalacja zakończona!"
echo "Aby uruchomić aplikację, wykonaj: cd $APP_DIR && ./restart.sh" 