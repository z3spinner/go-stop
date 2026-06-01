#!/bin/bash
set -e
for f in $(ls /migrations/*.up.sql 2>/dev/null | sort); do
    echo "Applying $f..."
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" < "$f"
done
