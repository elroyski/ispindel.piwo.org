#!/bin/bash
# =============================================================================
# Skrypt uruchamiający aplikację iSpindel
# =============================================================================

# Konfiguracja ścieżek
APP_DIR="/home/piwo/domains/ispindel.piwo.org/ispindle-web"
APP_NAME="ispindel"
LOG_FILE="$APP_DIR/app.log"
ENV_FILE="$APP_DIR/.env"

# Funkcja do wyświetlania wiadomości
log_message() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Funkcja do obsługi błędów
handle_error() {
    log_message "BŁĄD: $1"
    exit 1
}

# Sprawdzenie katalogu
log_message "Rozpoczynam uruchamianie aplikacji..."
if [ "$(pwd)" != "$APP_DIR" ]; then
    log_message "Przechodzę do katalogu $APP_DIR..."
    cd "$APP_DIR" || handle_error "Nie mogę przejść do katalogu $APP_DIR"
fi

# Sprawdzenie pliku wykonywalnego
if [ ! -f "./$APP_NAME" ]; then
    handle_error "Plik wykonywalny $APP_NAME nie istnieje w $APP_DIR"
fi

# Sprawdzenie i utworzenie pliku .env jeśli nie istnieje
if [ ! -f "$ENV_FILE" ]; then
    log_message "Tworzę plik konfiguracyjny .env..."
    cat > "$ENV_FILE" << EOF
# Konfiguracja bazy danych MySQL
DB_HOST=mysql18.mydevil.net
DB_USER=m1270_ispindel
DB_PASSWORD=Kochanapysia1
DB_NAME=m1270_ispindel
DB_PORT=3306

# Konfiguracja JWT
JWT_SECRET=twoj-tajny-klucz-jwt

# Konfiguracja aplikacji
PORT=49330
APP_URL=https://ispindel.piwo.org

# Konfiguracja SMTP
SMTP_HOST=mail18.mydevil.net
SMTP_PORT=587
SMTP_USER=ispindel@piwo.org
SMTP_PASSWORD=Kochanapysia1
SMTP_FROM=ispindel@piwo.org
EOF
    log_message "Utworzono plik .env z domyślnymi ustawieniami"
fi

# Eksportowanie zmiennych środowiskowych z pliku .env
log_message "Ładowanie zmiennych środowiskowych z pliku .env..."
export $(grep -v '^#' "$ENV_FILE" | xargs)

# Wyświetlenie ważnych zmiennych środowiskowych dla debugowania
log_message "Używam następujących ustawień:"
log_message "- Baza danych: $DB_HOST:$DB_PORT ($DB_NAME)"
log_message "- Port aplikacji: $PORT"
log_message "- URL aplikacji: $APP_URL"

# Zatrzymanie poprzedniej instancji
log_message "Zatrzymywanie poprzedniej instancji aplikacji..."
pkill -f "$APP_NAME" || log_message "Brak działającej instancji do zatrzymania"

# Uruchomienie aplikacji
log_message "Uruchamianie nowej instancji aplikacji..."
nohup ./"$APP_NAME" > "$LOG_FILE" 2>&1 &

# Sprawdzenie czy aplikacja uruchomiła się poprawnie
sleep 2
if pgrep -f "$APP_NAME" > /dev/null; then
    log_message "Aplikacja uruchomiona pomyślnie!"
    log_message "Logi zapisywane do pliku: $LOG_FILE"
    log_message "Aby sprawdzić logi, użyj: tail -f $LOG_FILE"
else
    log_message "Aplikacja nie uruchomiła się poprawnie. Sprawdź logi:"
    tail -n 20 "$LOG_FILE"
    handle_error "Nie udało się uruchomić aplikacji"
fi

log_message "Proces uruchamiania zakończony pomyślnie" 