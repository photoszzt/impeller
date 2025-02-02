module sharedlog-stream

go 1.22

toolchain go1.22.3

require (
	cs.utexas.edu/zjia/faas v0.0.0
	github.com/Jeffail/gabs/v2 v2.6.1
	github.com/brianvoe/gofakeit/v7 v7.0.3
	github.com/cespare/xxhash/v2 v2.2.0
	github.com/confluentinc/confluent-kafka-go v1.8.2
	github.com/gammazero/deque v0.2.1
	github.com/go-redis/redis/v9 v9.0.0-beta.2
	github.com/google/btree v1.1.2
	github.com/google/go-cmp v0.6.0
	github.com/minio/minio-go/v7 v7.0.67
	github.com/rs/zerolog v1.26.1
	github.com/spaolacci/murmur3 v1.1.0
	github.com/stretchr/testify v1.8.0
	github.com/tinylib/msgp v1.1.6
	github.com/zhangyunhao116/skipmap v0.0.0
	github.com/zhangyunhao116/skipset v0.12.1
	golang.org/x/sync v0.6.0
	golang.org/x/tools v0.6.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/grpc v1.63.2
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/zhangyunhao116/fastrand v0.3.0 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/petermattis/goid v0.0.0-20180202154549-b0b1615b78e5 // indirect
	github.com/philhofer/fwd v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sasha-s/go-deadlock v0.3.1
	golang.org/x/exp v0.0.0-20220722155223-a9213eeb770e
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace cs.utexas.edu/zjia/faas => ../boki/worker/golang

replace cs.utexas.edu/zjia/faas/slib => ../boki/faas/slib

replace github.com/zhangyunhao116/skipmap v0.0.0 => github.com/photoszzt/skipmap v0.0.0-20221220072452-3dc3cdf11b4c
