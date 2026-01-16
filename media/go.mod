module microservices/media

go 1.22

require (
	github.com/disintegration/imaging v1.6.2
	github.com/go-chi/chi/v5 v5.0.12
	github.com/google/uuid v1.6.0
	github.com/minio/minio-go/v7 v7.0.70
	github.com/nats-io/nats.go v1.34.0
	github.com/redis/go-redis/v9 v9.7.0
	microservices/pkg/authorization v0.0.0
	microservices/pkg/cache v0.0.0
)

replace (
	microservices/pkg/authorization => ../pkg/authorization
	microservices/pkg/cache => ../pkg/cache
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/rs/xid v1.5.0 // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/image v0.15.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)
