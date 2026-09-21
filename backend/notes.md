goose -dir internal/migrate/postgres/migrations postgres "postgresql://postgres:postgres@127.0.0.1:5436/issue_track?sslmode=disable" down
goose -dir internal/migrate/postgres/migrations create new_table sql
scp -r ./dist administrator@route:/home/administrator/apps/issue_track
npx vite-bundle-visualizer

название ITS (Issue Tracking System) или IssueTrack (трекер инцидентов)

export DOCKER_API_VERSION=1.44

# mattermost.allowed_server_ips (config.yaml, env MATTERMOST_ALLOWED_SERVER_IPS)
# Разрешённые источники входящих /mattermost/* колбэков (сам MM или его прокси).
# Форматы записи:
#   - одиночный IP:      "192.168.5.230"
#   - подсеть (CIDR):    "192.168.5.0/24"
#   - диапазон IP:       "192.168.5.200-192.168.5.255"
# Пустой список = все входящие MM-запросы отклоняются (403). Запись для
# "все адреса выше .200 в подсети 192.168.5.*": "192.168.5.200-192.168.5.255".
