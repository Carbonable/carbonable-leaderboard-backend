default:
    just --list

db_url := "postgres://carbonable:carbonable@localhost:5432/carbonable_leaderboard?sslmode=disable"
feeder_gateway := "https://alpha-sepolia.starknet.io/feeder_gateway"

# start docker database
start_db:
    docker compose -f docker-compose.yml up -d

# stop docker database
stop_db:
    docker compose -f docker-compose.yml stop

# run synchronizer application
sync:
    FEEDER_GATEWAY={{feeder_gateway}} DATABASE_URL={{db_url}} go run cmd/synchronizer/main.go

# run synchronizer application
sync_clean: stop_db start_db migrate
    DATABASE_URL={{db_url}} go run cmd/synchronizer/main.go

# run indexer application
indexer:
    FEEDER_GATEWAY={{feeder_gateway}} DATABASE_URL={{db_url}} go run cmd/indexer/main.go

# run api
api:
    DATABASE_URL={{db_url}} go run cmd/api/main.go

# run aggregator
aggregate:
    DATABASE_URL={{db_url}} go run cmd/aggregator/main.go

# run migrations
migrate:
    DATABASE_URL={{db_url}} go run cmd/migration/main.go

# run migrations
migrate_fresh:
    DATABASE_URL={{db_url}} go run cmd/migration/main.go -fresh
