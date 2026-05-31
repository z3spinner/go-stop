web: go-stop
postdeploy: psql $DATABASE_URL < db/migrations/001_create_tables.sql && psql $DATABASE_URL < db/migrations/002_add_stats.sql
