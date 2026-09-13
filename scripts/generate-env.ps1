<#
.SYNOPSIS
    Generates synchronized .env configuration files for all NexusCommerce microservices.

.DESCRIPTION
    This script inspects every microservice in the repository and generates a synchronized .env file
    based on .env.example or configuration defaults.
    
    Supports two modes:
    - Host (default): Configures datastore hosts and ports to connect to WSL2 Docker mapped ports
      (Postgres: 15432, Redis: 16379, MinIO: 9002, ClickHouse: 9000/8123, RabbitMQ: 5672, NATS: 4222, MongoDB: 27017, ES: 9200)
    - Container: Configures internal Docker network hostnames (postgres:5432, redis:6379, minio:9000, etc.)

.PARAMETER Mode
    Either 'Host' or 'Container'. Defaults to 'Host'.

.PARAMETER Force
    If specified, overwrites existing .env files without confirmation.
#>

param(
    [ValidateSet("Host", "Container")]
    [string]$Mode = "Host",
    [switch]$Force
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " NexusCommerce - Automated Environment Configuration ($Mode Mode)" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

# Common Credentials & Secrets
$Common = @{
    JwtAccessSecret   = "nexus_enterprise_super_secret_jwt_access_token_32chars_min"
    JwtRefreshSecret  = "nexus_enterprise_super_secret_jwt_refresh_token_32chars_min"
    InternalKey       = "nexus_internal_service_to_service_shared_secret_key_2026"
    PostgresUser      = "tafu_admin"
    PostgresPassword  = "tafu_secret_2024"
    RedisPassword     = "tafu_redis_2024"
    MongoUser         = "tafu_admin"
    MongoPassword     = "tafu_mongo_2024"
    ClickHouseUser    = "tafu_admin"
    ClickHousePassword= "tafu_click_2024"
    MinioUser         = "tafu_admin"
    MinioPassword     = "tafu_minio_2024"
    RabbitUser        = "tafu_admin"
    RabbitPassword    = "tafu_rabbit_2024"
}

# Network connection settings per mode
if ($Mode -eq "Host") {
    $Net = @{
        PostgresHost = "localhost"
        PostgresPort = "15432"
        RedisHost    = "localhost"
        RedisPort    = "16379"
        MongoHost    = "localhost"
        MongoPort    = "27017"
        ClickHost    = "localhost"
        ClickPortNative = "9000"
        ClickPortHttp   = "8123"
        MinioHost    = "localhost"
        MinioPort    = "9002"
        RabbitHost   = "localhost"
        RabbitPort   = "5672"
        NatsHost     = "localhost"
        NatsPort     = "4222"
        ElasticHost  = "localhost"
        ElasticPort  = "9200"
    }
} else {
    $Net = @{
        PostgresHost = "postgres"
        PostgresPort = "5432"
        RedisHost    = "redis"
        RedisPort    = "6379"
        MongoHost    = "mongodb"
        MongoPort    = "27017"
        ClickHost    = "clickhouse"
        ClickPortNative = "9000"
        ClickPortHttp   = "8123"
        MinioHost    = "minio"
        MinioPort    = "9000"
        RabbitHost   = "rabbitmq"
        RabbitPort   = "5672"
        NatsHost     = "nats"
        NatsPort     = "4222"
        ElasticHost  = "elasticsearch"
        ElasticPort  = "9200"
    }
}

# Helper to write .env safely
function Set-EnvFile {
    param(
        [string]$FilePath,
        [string]$Content
    )
    if ((Test-Path $FilePath) -and -not $Force) {
        Write-Host "[-] Skipping existing $FilePath (use -Force to overwrite)" -ForegroundColor Yellow
        return
    }
    [System.IO.File]::WriteAllText($FilePath, $Content.Trim(), [System.Text.UTF8Encoding]::new($false))
    Write-Host "[+] Generated $FilePath" -ForegroundColor Green
}

# 1. deploy/compose/.env
$composeEnv = @"
VERSION=latest
POSTGRES_USER=$($Common.PostgresUser)
POSTGRES_PASSWORD=$($Common.PostgresPassword)
REDIS_PASSWORD=$($Common.RedisPassword)
MONGO_PASSWORD=$($Common.MongoPassword)
CLICKHOUSE_PASSWORD=$($Common.ClickHousePassword)
MINIO_PASSWORD=$($Common.MinioPassword)
RABBITMQ_PASSWORD=$($Common.RabbitPassword)
JWT_SECRET=$($Common.JwtAccessSecret)
JWT_REFRESH_SECRET=$($Common.JwtRefreshSecret)
INTERNAL_SERVICE_KEY=$($Common.InternalKey)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "deploy\compose\.env") -Content $composeEnv

# 2. auth/.env
$authEnv = @"
APP_NAME=tafu-auth
NODE_ENV=development
PORT=3001
API_PREFIX=api/v1
CORS_ORIGIN=*
DEBUG=true

# Database (PostgreSQL)
DB_HOST=$($Net.PostgresHost)
DB_PORT=$($Net.PostgresPort)
DB_USERNAME=$($Common.PostgresUser)
DB_PASSWORD=$($Common.PostgresPassword)
DB_DATABASE=identity_db
DB_SSL_MODE=disable
DB_MAX_IDLE_CONNS=10
DB_MAX_OPEN_CONNS=100
DB_CONN_MAX_LIFETIME=1h
DB_LOG_LEVEL=warn

# Redis
REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
REDIS_DB=0

# JWT
JWT_ACCESS_SECRET=$($Common.JwtAccessSecret)
JWT_REFRESH_SECRET=$($Common.JwtRefreshSecret)
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# Cookie
COOKIE_DOMAIN=
COOKIE_SECURE=false
COOKIE_SAMESITE=Lax

# NATS
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "auth\.env") -Content $authEnv

# 2. profile/.env
$profileEnv = @"
SERVER_PORT=3002
PORT=3002
DB_HOST=$($Net.PostgresHost)
DB_PORT=$($Net.PostgresPort)
DB_USER=$($Common.PostgresUser)
DB_PASSWORD=$($Common.PostgresPassword)
DB_NAME=profile_db
DB_SSLMODE=disable

MONGO_URI=mongodb://$($Common.MongoUser):$($Common.MongoPassword)@$($Net.MongoHost):$($Net.MongoPort)/profile_db?authSource=admin
JWT_SECRET=$($Common.JwtAccessSecret)
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "profile\.env") -Content $profileEnv

# 3. catalog/.env
$catalogEnv = @"
SERVER_PORT=3003
PORT=3003
APP_PORT=3003
APP_GRPC_PORT=50053
MONGO_URI=mongodb://$($Common.MongoUser):$($Common.MongoPassword)@$($Net.MongoHost):$($Net.MongoPort)/catalog_db?authSource=admin
MONGODB_URI=mongodb://$($Common.MongoUser):$($Common.MongoPassword)@$($Net.MongoHost):$($Net.MongoPort)/catalog_db?authSource=admin
MONGODB_DATABASE=catalog_db
REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_ADDR=$($Net.RedisHost):$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "catalog\.env") -Content $catalogEnv

# 4. cart/.env
$cartEnv = @"
SERVER_PORT=3004
PORT=3004
REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
MONGO_URI=mongodb://$($Common.MongoUser):$($Common.MongoPassword)@$($Net.MongoHost):$($Net.MongoPort)/cart_db?authSource=admin
JWT_SECRET=$($Common.JwtAccessSecret)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "cart\.env") -Content $cartEnv

# 5. order/.env
$orderEnv = @"
SERVER_HOST=0.0.0.0
SERVER_PORT=3005
PORT=3005

DB_HOST=$($Net.PostgresHost)
DB_PORT=$($Net.PostgresPort)
DB_USER=$($Common.PostgresUser)
DB_PASSWORD=$($Common.PostgresPassword)
DB_NAME=order_db
DB_SSLMODE=disable

NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
NATS_STREAM_NAME=ORDERS

REDIS_URL=$($Net.RedisHost):$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)

INVENTORY_GRPC_ADDRESS=localhost:50052
INVENTORY_TIMEOUT=5
PAYMENT_SERVICE_URL=http://localhost:3007
"@
Set-EnvFile -FilePath (Join-Path $RootDir "order\.env") -Content $orderEnv

# 6. inventory/.env
$invEnv = @"
SERVER_PORT=3006
PORT=3006
GRPC_PORT=50052
INVENTORY_APP_PORT=3006
INVENTORY_APP_GRPC_PORT=50052

DB_HOST=$($Net.PostgresHost)
DB_PORT=$($Net.PostgresPort)
DB_USER=$($Common.PostgresUser)
DB_PASSWORD=$($Common.PostgresPassword)
DB_NAME=inventory_db
DB_SSLMODE=disable

INVENTORY_POSTGRES_HOST=$($Net.PostgresHost)
INVENTORY_POSTGRES_PORT=$($Net.PostgresPort)
INVENTORY_POSTGRES_USER=$($Common.PostgresUser)
INVENTORY_POSTGRES_PASSWORD=$($Common.PostgresPassword)
INVENTORY_POSTGRES_DATABASE=inventory_db
INVENTORY_POSTGRES_SSL_MODE=disable

REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
INVENTORY_REDIS_ADDR=$($Net.RedisHost):$($Net.RedisPort)
INVENTORY_REDIS_PASSWORD=$($Common.RedisPassword)

NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "inventory\.env") -Content $invEnv

# 7. payment/.env
$payEnv = @"
SERVER_PORT=3007
PORT=3007
DATABASE_URL=postgres://$($Common.PostgresUser):$($Common.PostgresPassword)@$($Net.PostgresHost):$($Net.PostgresPort)/payment_db?sslmode=disable
DB_HOST=$($Net.PostgresHost)
DB_PORT=$($Net.PostgresPort)
DB_USER=$($Common.PostgresUser)
DB_PASSWORD=$($Common.PostgresPassword)
DB_NAME=payment_db
DB_SSLMODE=disable

REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "payment\.env") -Content $payEnv

# 8. logistic/.env
$logEnv = @"
SERVER_PORT=3008
PORT=3008
DATABASE_URL=postgres://$($Common.PostgresUser):$($Common.PostgresPassword)@$($Net.PostgresHost):$($Net.PostgresPort)/logistics_db?sslmode=disable
DB_HOST=$($Net.PostgresHost)
DB_PORT=$($Net.PostgresPort)
DB_USER=$($Common.PostgresUser)
DB_PASSWORD=$($Common.PostgresPassword)
DB_NAME=logistics_db
DB_SSLMODE=disable

REDIS_URL=redis://:$($Common.RedisPassword)@$($Net.RedisHost):$($Net.RedisPort)/0
REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "logistic\.env") -Content $logEnv

# 9. campaign/.env
$campEnv = @"
SERVER_PORT=3009
PORT=3009
APP_PORT=3009
APP_GRPC_PORT=50059
POSTGRES_HOST=$($Net.PostgresHost)
POSTGRES_PORT=$($Net.PostgresPort)
POSTGRES_USER=$($Common.PostgresUser)
POSTGRES_PASSWORD=$($Common.PostgresPassword)
POSTGRES_DATABASE=campaign_db
POSTGRES_SSL_MODE=disable

DB_HOST=$($Net.PostgresHost)
DB_PORT=$($Net.PostgresPort)
DB_USER=$($Common.PostgresUser)
DB_PASSWORD=$($Common.PostgresPassword)
DB_NAME=campaign_db
DB_SSLMODE=disable

REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "campaign\.env") -Content $campEnv

# 10. notification/.env
$notifEnv = @"
SERVER_PORT=3010
PORT=3010
APP_PORT=3010
MONGO_URI=mongodb://$($Common.MongoUser):$($Common.MongoPassword)@$($Net.MongoHost):$($Net.MongoPort)/notification_db?authSource=admin
MONGODB_URI=mongodb://$($Common.MongoUser):$($Common.MongoPassword)@$($Net.MongoHost):$($Net.MongoPort)/notification_db?authSource=admin
MONGODB_DATABASE=notification_db
RABBITMQ_URL=amqp://$($Common.RabbitUser):$($Common.RabbitPassword)@$($Net.RabbitHost):$($Net.RabbitPort)/
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "notification\.env") -Content $notifEnv

# 11. analytic/.env
$anaEnv = @"
SERVER_PORT=3014
PORT=3014
CLICKHOUSE_HOST=$($Net.ClickHost)
CLICKHOUSE_PORT=$($Net.ClickPortNative)
CLICKHOUSE_DATABASE=analytics
CLICKHOUSE_USERNAME=$($Common.ClickHouseUser)
CLICKHOUSE_USER=$($Common.ClickHouseUser)
CLICKHOUSE_PASSWORD=$($Common.ClickHousePassword)
RABBITMQ_URL=amqp://$($Common.RabbitUser):$($Common.RabbitPassword)@$($Net.RabbitHost):$($Net.RabbitPort)/
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "analytic\.env") -Content $anaEnv

# 12. media/.env
$mediaEnv = @"
SERVER_PORT=3011
PORT=3011
APP_PORT=3011
MINIO_ENDPOINT=$($Net.MinioHost):$($Net.MinioPort)
MINIO_ACCESS_KEY_ID=$($Common.MinioUser)
MINIO_SECRET_ACCESS_KEY=$($Common.MinioPassword)
MINIO_ACCESS_KEY=$($Common.MinioUser)
MINIO_SECRET_KEY=$($Common.MinioPassword)
MINIO_USE_SSL=false
MINIO_BUCKET_NAME=nexus-media
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "media\.env") -Content $mediaEnv

# 13. review/.env
$revEnv = @"
SERVER_PORT=3012
PORT=3012
APP_PORT=3012
MONGO_URI=mongodb://$($Common.MongoUser):$($Common.MongoPassword)@$($Net.MongoHost):$($Net.MongoPort)/review_db?authSource=admin
MONGODB_URI=mongodb://$($Common.MongoUser):$($Common.MongoPassword)@$($Net.MongoHost):$($Net.MongoPort)/review_db?authSource=admin
REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
JWT_SECRET=$($Common.JwtAccessSecret)
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "review\.env") -Content $revEnv

# 14. search/.env
$searchEnv = @"
SERVER_PORT=3013
PORT=3013
APP_PORT=3013
API_PORT=3013
ELASTICSEARCH_URL=http://$($Net.ElasticHost):$($Net.ElasticPort)
ELASTIC_ADDRESSES=http://$($Net.ElasticHost):$($Net.ElasticPort)
REDIS_ADDR=$($Net.RedisHost):$($Net.RedisPort)
REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "search\.env") -Content $searchEnv

# 17. realtime/.env
$realtimeEnv = @"
SERVER_PORT=3015
NATS_URL=nats://$($Net.NatsHost):$($Net.NatsPort)
JWT_SECRET=$($Common.JwtAccessSecret)
INTERNAL_SERVICE_KEY=$($Common.InternalKey)
REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "realtime\.env") -Content $realtimeEnv

# 18. api-gateway/.env
if ($Mode -eq "Host") {
    $gwIdentity    = "http://localhost:3001"
    $gwProfile     = "http://localhost:3002"
    $gwCatalog     = "http://localhost:3003"
    $gwCart        = "http://localhost:3004"
    $gwOrder       = "http://localhost:3005"
    $gwInventory   = "http://localhost:3006"
    $gwPayment     = "http://localhost:3007"
    $gwLogistic    = "http://localhost:3008"
    $gwCampaign    = "http://localhost:3009"
    $gwNotification= "http://localhost:3010"
    $gwMedia       = "http://localhost:3011"
    $gwReview      = "http://localhost:3012"
    $gwSearch      = "http://localhost:3013"
    $gwAnalytic    = "http://localhost:3014"
    $gwRealtime    = "http://localhost:3015"
} else {
    $gwIdentity    = "http://identity-service:3000"
    $gwProfile     = "http://profile-service:3000"
    $gwCatalog     = "http://catalog-service:3000"
    $gwCart        = "http://cart-service:3000"
    $gwOrder       = "http://order-service:3000"
    $gwInventory   = "http://inventory-service:3000"
    $gwPayment     = "http://payment-service:3000"
    $gwLogistic    = "http://logistics-service:3000"
    $gwCampaign    = "http://campaign-service:3000"
    $gwNotification= "http://notification-service:3000"
    $gwMedia       = "http://media-service:3000"
    $gwReview      = "http://review-service:3000"
    $gwSearch      = "http://search-service:3000"
    $gwAnalytic    = "http://analytics-service:3000"
    $gwRealtime    = "http://realtime-service:3015"
}

$gatewayEnv = @"
PORT=8080
IDENTITY_URL=$gwIdentity
PROFILE_URL=$gwProfile
CATALOG_URL=$gwCatalog
CART_URL=$gwCart
ORDER_URL=$gwOrder
INVENTORY_URL=$gwInventory
PAYMENT_URL=$gwPayment
LOGISTIC_URL=$gwLogistic
CAMPAIGN_URL=$gwCampaign
NOTIFICATION_URL=$gwNotification
MEDIA_URL=$gwMedia
REVIEW_URL=$gwReview
SEARCH_URL=$gwSearch
ANALYTIC_URL=$gwAnalytic
REALTIME_URL=$gwRealtime

REDIS_HOST=$($Net.RedisHost)
REDIS_PORT=$($Net.RedisPort)
REDIS_PASSWORD=$($Common.RedisPassword)
JWT_SECRET=$($Common.JwtAccessSecret)
INTERNAL_SERVICE_KEY=$($Common.InternalKey)
"@
Set-EnvFile -FilePath (Join-Path $RootDir "api-gateway\.env") -Content $gatewayEnv

Write-Host "============================================================" -ForegroundColor Green
Write-Host " Successfully synchronized all microservice .env files!" -ForegroundColor Green
Write-Host "============================================================" -ForegroundColor Green
