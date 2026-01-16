module microservices/api-gateway

go 1.22

require (
	github.com/gofiber/fiber/v2 v2.52.5
	github.com/gofiber/storage/redis/v3 v3.1.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	gopkg.in/yaml.v3 v3.0.1
	microservices/pkg/authorization v0.0.0
	microservices/pkg/cache v0.0.0
)

replace (
	microservices/pkg/authorization => ../pkg/authorization
	microservices/pkg/cache => ../pkg/cache
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/philhofer/fwd v1.1.2 // indirect
	github.com/redis/go-redis/v9 v9.7.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/tinylib/msgp v1.1.8 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.52.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.68.1 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)
