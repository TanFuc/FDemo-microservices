module microservices/order

go 1.22

require (
	github.com/gofiber/fiber/v2 v2.52.5
	github.com/google/uuid v1.6.0
	github.com/nats-io/nats.go v1.37.0
	github.com/shopspring/decimal v1.4.0
	github.com/spf13/viper v1.18.2
	google.golang.org/grpc v1.68.1
	google.golang.org/protobuf v1.35.1
	gorm.io/driver/postgres v1.5.9
	gorm.io/gorm v1.25.12
	microservices/pkg/authorization v0.0.0
	microservices/pkg/authclient v0.0.0
	microservices/pkg/cache v0.0.0
	microservices/pkg/customfields v0.0.0
	microservices/pkg/logger v0.0.0
	microservices/pkg/response v0.0.0
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.1 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/redis/go-redis/v9 v9.7.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.55.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240924160255-9d4c2d233b61 // indirect
)

replace (
	microservices/pkg/authorization => ../pkg/authorization
	microservices/pkg/authclient => ../pkg/authclient
	microservices/pkg/cache => ../pkg/cache
	microservices/pkg/customfields => ../pkg/customfields
	microservices/pkg/logger => ../pkg/logger
	microservices/pkg/response => ../pkg/response
)
