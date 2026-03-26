goose -dir internal/migrate/postgres/migrations postgres "postgresql://postgres:postgres@127.0.0.1:5436/issue_track?sslmode=disable" down
goose -dir internal/migrate/postgres/migrations create new_table sql
scp -r ./dist administrator@route:/home/administrator/apps/issue_track
npx vite-bundle-visualizer

название ITS (Issue Tracking System) или IssueTrack (трекер инцидентов)

export DOCKER_API_VERSION=1.44
