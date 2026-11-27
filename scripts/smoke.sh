#!/usr/bin/env bash
# Дымовой тест: создать ссылку → найти в БД → проверить редирект → удалить.
set -euo pipefail

BASE="${BASE:-http://localhost:9091}"
ALIAS="smoke-$(date +%s)"
TARGET="https://example.com/smoke-test"
FAILED=0

sql() {
  docker compose exec -T db sh -c \
    'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tA -c "$1"' _ "$1"
}

# JSON собираем printf-ом: скобки и запятые остаются внутри одной строки
body() { # url [alias]
  if [ -n "${2:-}" ]; then
    printf '{"url":"%s","alias":"%s"}' "$1" "$2"
  else
    printf '{"url":"%s"}' "$1"
  fi
}

post_code() { # тело запроса → HTTP-код
  curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE/shorten" \
    -H 'Content-Type: application/json' --data "$1"
}

get_code() { curl -s -o /dev/null -w '%{http_code}' "$1"; }

expect() { # название, получено, ожидалось
  if [ "$#" -ne 3 ]; then
    echo "  FAIL  $1: ошибка скрипта, передано $# аргументов вместо 3"
    FAILED=1
    return
  fi
  if [ "$2" = "$3" ]; then
    echo "  ok    $1"
  else
    echo "  FAIL  $1: получено '$2', ожидалось '$3'"
    FAILED=1
  fi
}

cleanup() { sql "DELETE FROM url WHERE url_short = '$ALIAS'" >/dev/null 2>&1 || true; }
trap cleanup EXIT

echo "Жду готовности сервиса..."
curl -fsS --retry 20 --retry-delay 1 --retry-connrefused -o /dev/null "$BASE/readyz"

echo "1. Создание ссылки (alias=$ALIAS)"
expect "POST /shorten → 201" "$(post_code "$(body "$TARGET" "$ALIAS")")" 201

echo "2. Проверка в БД"
expect "строка есть в таблице" \
  "$(sql "SELECT count(*) FROM url WHERE url_short = '$ALIAS'")" 1
expect "адрес сохранён верно" \
  "$(sql "SELECT url_original FROM url WHERE url_short = '$ALIAS'")" "$TARGET"

echo "3. Редирект"
expect "GET /$ALIAS → 302 на оригинал" \
  "$(curl -s -o /dev/null -w '%{http_code} %{redirect_url}' "$BASE/$ALIAS")" "302 $TARGET"

echo "4. Ошибочные случаи"
expect "повтор алиаса → 409" "$(post_code "$(body "$TARGET" "$ALIAS")")" 409
expect "алиас с пробелом → 400" "$(post_code "$(body "$TARGET" "a b")")" 400
expect "зарезервированный алиас → 400" "$(post_code "$(body "$TARGET" "health")")" 400
expect "невалидный URL → 400" "$(post_code "$(body "not-a-url")")" 400
expect "битый JSON → 400" "$(post_code 'not json')" 400
expect "несуществующий код → 404" "$(get_code "$BASE/none-$ALIAS")" 404

echo "5. Удаление"
sql "DELETE FROM url WHERE url_short = '$ALIAS'" >/dev/null
expect "строки в БД больше нет" \
  "$(sql "SELECT count(*) FROM url WHERE url_short = '$ALIAS'")" 0
expect "GET /$ALIAS после удаления → 404" "$(get_code "$BASE/$ALIAS")" 404

echo
if [ "$FAILED" -eq 0 ]; then
  echo "ВСЕ ПРОВЕРКИ ПРОЙДЕНЫ"
else
  echo "ЕСТЬ ОШИБКИ"
  exit 1
fi