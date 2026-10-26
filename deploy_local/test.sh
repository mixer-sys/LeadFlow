#!/usr/bin/env bash
# Smoke-тесты LeadFlow API.
# Запуск: bash test.sh
# Зависимости: curl, jq.
#
# ВНИМАНИЕ:
# Создаёт лиды и удаляет первый созданный тестовый лид.
# При работающем worker возможны реальные Telegram/CRM/webhook-уведомления.

set -uo pipefail

if [[ -f .env ]]; then
  set -a
  source .env
  set +a
fi

BASE_URL="${BASE_URL:-http://localhost:8080}"
NGINX_URL="${NGINX_URL:-http://localhost}"
API_PREFIX="${API_PREFIX:-/api/v1}"
API_KEY="${API_KEY:-}"

BASE_URL="${BASE_URL%/}"
NGINX_URL="${NGINX_URL%/}"

for dependency in curl jq; do
  if ! command -v "$dependency" >/dev/null 2>&1; then
    echo "ERROR: не установлен $dependency" >&2
    exit 2
  fi
done

if [[ -z "$API_KEY" ]]; then
  echo "ERROR: API_KEY не задан. Добавь его в .env или окружение." >&2
  exit 2
fi

TMP_DIR="$(mktemp -d)" || exit 2
trap 'rm -rf "$TMP_DIR"' EXIT

RUN_ID="$(date +%s)-$$"
LEADS_URL="${BASE_URL}${API_PREFIX}/leads"

PASSED=0
FAILED=0
SKIPPED=0
FAILURES=()
SKIPS=()

LAST_BODY=""
LAST_OK=0
LEAD_ID=""

fail() {
  local message="$1"

  FAILED=$((FAILED + 1))
  FAILURES+=("$message")
  printf 'FAIL: %s\n' "$message"
}

skip() {
  local name="$1"
  local reason="$2"

  SKIPPED=$((SKIPPED + 1))
  SKIPS+=("$name — $reason")
  printf '\nSKIP: %s — %s\n' "$name" "$reason"
}

# Аргументы:
#   название, ожидаемые коды через пробел, метод, URL,
#   режим авторизации: valid / none / wrong,
#   дополнительные аргументы curl.
check() {
  local name="$1"
  local expected="$2"
  local method="$3"
  local url="$4"
  local auth="$5"
  shift 5

  local status
  local curl_rc=0
  local body_file="$TMP_DIR/body"
  local error_file="$TMP_DIR/error"
  local -a headers=(
    -H "Content-Type: application/json"
  )

  case "$auth" in
    valid)
      headers+=(-H "X-API-Key: ${API_KEY}")
      ;;
    wrong)
      headers+=(-H "X-API-Key: wrong-key")
      ;;
    none)
      ;;
    *)
      fail "$name — неизвестный режим авторизации: $auth"
      return 0
      ;;
  esac

  LAST_BODY=""
  LAST_OK=0

  : > "$body_file"
  : > "$error_file"

  printf '\n==================================================\n'
  printf '%s\n' "$name"
  printf '>>> %s %s\n' "$method" "$url"

  status="$(
    curl -sS \
      --connect-timeout 5 \
      --max-time 20 \
      -X "$method" \
      "${headers[@]}" \
      -o "$body_file" \
      -w '%{http_code}' \
      "$@" \
      "$url" \
      2>"$error_file"
  )" || curl_rc=$?

  LAST_BODY="$(cat "$body_file")"

  if [[ -n "$LAST_BODY" ]]; then
    if jq . "$body_file" >/dev/null 2>&1; then
      jq . "$body_file"
    else
      cat "$body_file"
      printf '\n'
    fi
  fi

  if (( curl_rc != 0 )); then
    fail "$name — ошибка curl $curl_rc: $(cat "$error_file")"
  elif [[ " $expected " == *" $status "* ]]; then
    PASSED=$((PASSED + 1))
    LAST_OK=1
    printf 'PASS: HTTP %s\n' "$status"
  else
    fail "$name — ожидался HTTP [$expected], получен HTTP $status"
  fi

  # Не останавливаем весь набор после проваленной проверки.
  return 0
}

check "1. Health check" \
  "200" GET "$BASE_URL/healthz" none

check "2. Create full lead" \
  "200 201" POST "$LEADS_URL" valid \
  --data "$(jq -n \
    --arg email "smoke+${RUN_ID}@example.com" \
    '{
      source: "manual",
      name: "Иван Петров",
      email: $email,
      phone: "+79990000001",
      message: "Тестовая заявка",
      status: "new"
    }')"

# Отдельная проверка: создание должно вернуть числовой ID.
if (( LAST_OK == 1 )); then
  LEAD_ID="$(
    jq -r '.id // empty' <<< "$LAST_BODY" 2>/dev/null
  )" || LEAD_ID=""

  if [[ "$LEAD_ID" =~ ^[0-9]+$ ]]; then
    PASSED=$((PASSED + 1))
    printf 'PASS: 3. Extract lead ID — %s\n' "$LEAD_ID"
  else
    LEAD_ID=""
    fail "3. Extract lead ID — в ответе нет числового поля .id"
  fi
else
  skip "3. Extract lead ID" "создание полного лида не прошло"
fi

check "4. Create minimal lead" \
  "200 201" POST "$LEADS_URL" valid \
  --data "$(jq -n \
    --arg email "maria+${RUN_ID}@example.com" \
    '{
      source: "manual",
      name: "Мария Смирнова",
      email: $email,
      message: "Заявка без телефона"
    }')"

check "5. Create Telegram lead" \
  "200 201" POST "$LEADS_URL" valid \
  --data "$(jq -n \
    --arg email "alexey+${RUN_ID}@example.com" \
    '{
      source: "telegram",
      name: "Алексей Иванов",
      email: $email,
      phone: "+79990000003",
      message: "Заявка из Telegram",
      status: "new"
    }')"

check "6. List leads" \
  "200" GET "$LEADS_URL" valid

check "7. List leads with pagination" \
  "200" GET "$LEADS_URL?limit=10&offset=0" valid

check "8. Filter leads by status" \
  "200" GET "$LEADS_URL?status=new" valid

check "9. Filter leads by source" \
  "200" GET "$LEADS_URL?source=manual" valid

if [[ -n "$LEAD_ID" ]]; then
  check "10. Get lead by ID" \
    "200" GET "$LEADS_URL/$LEAD_ID" valid

  check "11. Update lead status" \
    "200 204" PATCH "$LEADS_URL/$LEAD_ID" valid \
    --data '{"status":"processed"}'

  check "12. Delete lead" \
    "200 204" DELETE "$LEADS_URL/$LEAD_ID" valid
else
  skip "10. Get lead by ID" "LEAD_ID не получен"
  skip "11. Update lead status" "LEAD_ID не получен"
  skip "12. Delete lead" "LEAD_ID не получен"
fi

check "13. Unauthorized: no API key" \
  "401" GET "$LEADS_URL" none

check "14. Unauthorized: wrong API key" \
  "401" GET "$LEADS_URL" wrong

check "15. Validation error: empty body" \
  "400" POST "$LEADS_URL" valid \
  --data '{}'

check "16. Validation error: invalid email" \
  "400" POST "$LEADS_URL" valid \
  --data '{
    "source": "manual",
    "name": "Bad Email",
    "email": "not-an-email",
    "message": "Validation test"
  }'

check "17. Validation error: invalid JSON" \
  "400" POST "$LEADS_URL" valid \
  --data '{ invalid json }'

# При наличии удалённого ID проверяем его,
# иначе используем ID из исходного скрипта.
NOT_FOUND_ID="${LEAD_ID:-999999}"

check "18. Not found" \
  "404" GET "$LEADS_URL/$NOT_FOUND_ID" valid

check "19. Method not allowed" \
  "405" PUT "$LEADS_URL" valid

# RUN_NGINX_TEST=0 — для локальной разработки без nginx.
if [[ "${RUN_NGINX_TEST:-1}" == "1" ]]; then
  check "20a. Health through nginx" \
    "200" GET "$NGINX_URL/healthz" none
else
  skip "20a. Health through nginx" "RUN_NGINX_TEST=0"
fi

DIRECT_API_URL="${DIRECT_API_URL:-http://localhost:8080}"

check "20b. Direct API health" \
  "200" GET "${DIRECT_API_URL%/}/healthz" none

printf '\n==================================================\n'
printf 'РЕЗУЛЬТАТЫ\n'
printf '==================================================\n'
printf 'Успешно:  %d\n' "$PASSED"
printf 'Провалено: %d\n' "$FAILED"
printf 'Пропущено: %d\n' "$SKIPPED"

if (( FAILED > 0 )); then
  printf '\nНе прошли:\n'
  for item in "${FAILURES[@]}"; do
    printf '  - %s\n' "$item"
  done
fi

if (( SKIPPED > 0 )); then
  printf '\nПропущены:\n'
  for item in "${SKIPS[@]}"; do
    printf '  - %s\n' "$item"
  done
fi

printf '\n'

if (( FAILED > 0 )); then
  printf 'ИТОГ: НЕ ОК — успешно %d, провалено %d, пропущено %d.\n' \
    "$PASSED" "$FAILED" "$SKIPPED"
  exit 1
elif (( SKIPPED > 0 )); then
  printf 'ИТОГ: все выполненные проверки прошли, но %d пропущено.\n' \
    "$SKIPPED"
  exit 0
else
  printf 'ИТОГ: ВСЁ ОК — все %d проверок прошли.\n' "$PASSED"
  exit 0
fi