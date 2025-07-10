module github.com/memtensor/memgos

go 1.24

require (
	github.com/PuerkitoBio/goquery v1.9.3
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/cenkalti/backoff/v4 v4.2.1
	// Core dependencies
	github.com/fsnotify/fsnotify v1.7.0
	github.com/go-playground/validator/v10 v10.26.0

	// HTTP client dependencies for LLM integration
	github.com/go-resty/resty/v2 v2.7.0

	// User management dependencies
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0

	// Text processing dependencies
	github.com/ledongthuc/pdf v0.0.0-20220302134840-0c2507a12d80

	// NATS dependencies for distributed scheduling
	github.com/nats-io/nats.go v1.36.0

	// Graph database dependencies
	github.com/neo4j/neo4j-go-driver/v5 v5.15.0
	github.com/sashabaranov/go-openai v1.17.9

	// Configuration and validation
	github.com/spf13/viper v1.18.2
	github.com/stretchr/testify v1.10.0
	github.com/yuin/goldmark v1.6.0
	golang.org/x/crypto v0.39.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.30.0
)

require (
	github.com/gin-contrib/cors v1.7.6
	github.com/gin-gonic/gin v1.10.1
	github.com/qdrant/go-client v1.14.1
	github.com/redis/go-redis/v9 v9.11.0
	google.golang.org/grpc v1.73.0
)

require (
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/bytedance/sonic v1.13.3 // indirect
	github.com/bytedance/sonic/loader v0.2.4 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.9 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.7 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.28 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/arch v0.18.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)
