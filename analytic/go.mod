module microservices/analytic

go 1.22

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.23.0
	github.com/google/uuid v1.6.0
	github.com/nats-io/nats.go v1.31.0
	github.com/redis/go-redis/v9 v9.7.0
	microservices/pkg/authorization v0.0.0
	microservices/pkg/cache v0.0.0
)

replace (
	microservices/pkg/authorization => ../pkg/authorization
	microservices/pkg/cache => ../pkg/cache
)
