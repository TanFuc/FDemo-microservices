#!/bin/bash
# Create all service databases before consolidated schemas are applied.

set -euo pipefail

create_database() {
    local database="$1"

    echo "Creating database '$database'..."
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
        CREATE DATABASE "$database";
        GRANT ALL PRIVILEGES ON DATABASE "$database" TO "$POSTGRES_USER";
EOSQL
}

if [[ -n "${POSTGRES_MULTIPLE_DATABASES:-}" ]]; then
    echo "Multiple database creation requested: $POSTGRES_MULTIPLE_DATABASES"
    IFS=',' read -ra databases <<< "$POSTGRES_MULTIPLE_DATABASES"
    for database in "${databases[@]}"; do
        create_database "$database"
    done
    echo "Multiple databases created."
fi
