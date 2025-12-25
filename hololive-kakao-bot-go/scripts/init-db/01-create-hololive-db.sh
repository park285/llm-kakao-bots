#!/bin/bash
set -e

# Create hololive database for hololive-kakao-bot-go
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE hololive;
    GRANT ALL PRIVILEGES ON DATABASE hololive TO $POSTGRES_USER;
EOSQL

echo "✅ Database 'hololive' created successfully"
