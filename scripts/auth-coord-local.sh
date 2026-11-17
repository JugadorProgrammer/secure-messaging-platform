#!/usr/bin/env bash
set -euo pipefail

if [[ $# != 1 ]]; then
    printf 'Использование: %s /путь/к/рабочему-дереву-COORD\n' "$0" >&2
    exit 2
fi

auth_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
coord_dir=$(cd -- "$1" && pwd -P)
if [[ ! -f "$coord_dir/docker-compose.yml" || "$auth_dir" == "$coord_dir" ]]; then
    printf 'Укажите отдельное рабочее дерево COORD с docker-compose.yml.\n' >&2
    exit 2
fi

for dependency in docker curl jq; do
    if ! command -v "$dependency" >/dev/null; then
        printf 'Требуется %s.\n' "$dependency" >&2
        exit 1
    fi
done
docker compose version >/dev/null

auth_url=http://127.0.0.1:8081
coord_url=http://127.0.0.1:8080
mailpit_url=http://127.0.0.1:8025

auth_compose() (
    cd -- "$auth_dir"
    AUTH_HTTP_HOST_PORT=8081 AUTH_POSTGRES_HOST_PORT=5433 \
        docker compose -f compose.yaml -f compose.mailpit.yaml "$@"
)

coord_compose() (
    cd -- "$coord_dir"
    HTTP_PORT=8080 POSTGRES_PORT=5432 ACCOUNTS_URL=http://host.docker.internal:8081 \
        docker compose -f docker-compose.yml "$@"
)

if [[ ! -e "$auth_dir/.env" ]]; then
    (umask 077; set -o noclobber; cat "$auth_dir/.env.example" > "$auth_dir/.env")
fi

printf 'Запуск AUTH, PostgreSQL и Mailpit…\n'
auth_compose up -d --build
printf 'Запуск COORD и его PostgreSQL…\n'
coord_compose up -d --build

http() {
    curl --noproxy 127.0.0.1 --silent --show-error --max-time 20 "$@"
}

wait_for_http() {
    local address=$1 deadline=$((SECONDS + 90))
    until http --max-time 2 --fail "$address" >/dev/null 2>&1; do
        if (( SECONDS >= deadline )); then
            printf 'Не дождались ответа %s. Проверьте docker compose logs.\n' "$address" >&2
            return 1
        fi
        sleep 1
    done
}

wait_for_http "$auth_url/health"
wait_for_http "$coord_url/health"
wait_for_http "$mailpit_url/api/v1/info"

work=$(mktemp -d)
suffix=${work##*.}
login=auth24_${suffix,,}
email="$login@example.test"
password='LocalDemoAuth24!'
created=false

cleanup() {
    local status=$?
    if [[ "$created" == true ]] &&
        ! printf '%s\n' "DELETE FROM users WHERE login = :'test_login' AND email = :'test_email';" |
            auth_compose exec -T postgres sh -c \
                'exec psql -q -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v "test_login=$1" -v "test_email=$2"' \
                sh "$login" "$email"; then
        printf 'Не удалось удалить тестовый аккаунт %s.\n' "$login" >&2
        if (( status == 0 )); then
            status=1
        fi
    fi
    rm -rf -- "$work"
    exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
touch "$work/cookies"

post() {
    local path=$1 payload=$2 expected=$3 status
    status=$(http -b "$work/cookies" -c "$work/cookies" -o "$work/response.json" \
        -w '%{http_code}' -H 'Content-Type: application/json' -d "$payload" "$coord_url$path")
    if [[ "$status" != "$expected" ]]; then
        printf 'POST %s: ожидается HTTP %s, получен %s.\n' "$path" "$expected" "$status" >&2
        return 1
    fi
}

cookie() {
    awk -v name="$1" '$6 == name { print $7 }' "$work/cookies"
}

wait_for_verification_code() {
    local deadline=$((SECONDS + 30)) id code
    while (( SECONDS < deadline )); do
        if id=$(http --max-time 3 --fail --get --data-urlencode "query=to:$email" "$mailpit_url/api/v1/search" |
            jq -er --arg email "$email" '
                first(.messages[]? | select(any(.To[]?; .Address == $email) and
                    ((.Subject // "") | test("verification"; "i"))) | .ID)
            '); then
            code=$(http --fail "$mailpit_url/api/v1/message/$id" |
                jq -er '.Text | capture("code is\\s+(?<code>[0-9]{6})"; "i").code')
            printf '%s\n' "$code"
            return
        fi
        sleep 1
    done
    printf 'Письмо с кодом входа не появилось в Mailpit.\n' >&2
    return 1
}

signup=$(jq -nc --arg login "$login" --arg email "$email" --arg password "$password" \
    '{login: $login, email: $email, password: $password}')
post /api/auth/signup "$signup" 201
created=true
recovery_code=$(jq -er '
    select(type == "array" and length == 10 and (unique | length) == 10 and
        all(.[]; type == "string" and length > 0)) | .[0]
' "$work/response.json")
printf 'Регистрация через COORD: HTTP 201 и 10 кодов.\n'

credentials=$(jq -nc --arg login "$login" --arg password "$password" '{login: $login, password: $password}')
post /api/auth/login "$credentials" 204
[[ -n "$(cookie auth_pending)" && -z "$(cookie session)" ]] || {
    printf 'После пароля ожидалась cookie auth_pending без полной сессии.\n' >&2
    exit 1
}

email_code=$(wait_for_verification_code)
post /api/auth/login/email-code "$(jq -nc --arg code "$email_code" '{code: $code}')" 204
[[ -n "$(cookie auth_recovery_pending)" && -z "$(cookie auth_pending)" && -z "$(cookie session)" ]] || {
    printf 'После кода из письма ожидалась только cookie auth_recovery_pending.\n' >&2
    exit 1
}

post /api/auth/login/recovery-code "$(jq -nc --arg code "$recovery_code" '{code: $code}')" 204
session=$(cookie session)
[[ -n "$session" && -z "$(cookie auth_recovery_pending)" ]] || {
    printf 'После резервного кода ожидалась полная сессия.\n' >&2
    exit 1
}

status=$(http -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $session" "$coord_url/api/users")
if [[ "$status" != 403 ]]; then
    printf 'COORD должен распознать сессию user и вернуть HTTP 403 для /api/users, получен %s.\n' "$status" >&2
    exit 1
fi
printf 'Трёхшаговый вход через COORD: сессия user получена, доступ к /api/users ограничен.\n'
printf 'AUTH: %s\nCOORD: %s\nMailpit: %s\n' "$auth_url" "$coord_url" "$mailpit_url"
