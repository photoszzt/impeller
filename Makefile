GO_FILES?=$$(find . -name '*.go' |grep -v deps)

default: lat_tp_handler_debug lat_tp_handler sharedlog_bench_client dump_stream \
	nexmark_kafka_test_src kafka_consume_bench kafka_produce_bench nexmark \
	nexmark_stats nexmark_debug nexmark_gen_data_by_spec nexmark_client \
	nexmark_genevents_kafka nexmark_scale kafka_tran_process sharedlog_protocol_lat \
	remote_txn_mngr_grpc remote_txn_mngr_grpc_debug remote_txn_mngr_boki remote_txn_mngr_boki_debug \
	tests_client tests_handler chkptmngr_server chkptmngr_server_debug

.PHONY: golangci-lint
golangci-lint:
	@golangci-lint -E goimports run --build-tags "debug,stats" ./...

.PHONY: nexmark_gen_data_by_spec
nexmark_gen_data_by_spec:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/nexmark_gen_data_by_spec ./benchmark/nexmark/gen_test_data

.PHONY: dump_stream
dump_stream:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/dump_stream ./benchmark/nexmark/dump_stream

.PHONY: nexmark
nexmark:
	mkdir -p ./bin
	GO111MODULE=on go build -tags "stats,printstats" -o bin/nexmark_handler ./benchmark/nexmark/nexmark_handler

.PHONY: tests_client
tests_client:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/tests_client ./benchmark/tests/tests_client/

.PHONY: tests_handler
tests_handler:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/tests_handler ./benchmark/tests/tests_handler/

.PHONY: nexmark_stats
nexmark_stats:
	mkdir -p ./bin
	GO111MODULE=on go build -tags stats -o bin/nexmark_handler_stats ./benchmark/nexmark/nexmark_handler

.PHONY: nexmark_debug
nexmark_debug:
	mkdir -p ./bin
	GO111MODULE=on go build -race -tags "debug,stats" -o bin/nexmark_handler_debug ./benchmark/nexmark/nexmark_handler

.PHONY: nexmark_client
nexmark_client:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/nexmark_client ./benchmark/nexmark/nexmark_client

.PHONY: remote_txn_mngr_grpc
remote_txn_mngr_grpc:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/remote_txn_mngr_grpc ./remote_txn_manager/grpc

.PHONY: remote_txn_mngr_grpc_debug
remote_txn_mngr_grpc_debug:
	mkdir -p ./bin
	GO111MODULE=on go build -race -tags "debug" -o bin/remote_txn_mngr_grpc_debug ./remote_txn_manager/grpc

.PHONY: remote_txn_mngr_boki
remote_txn_mngr_boki:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/remote_txn_mngr_boki ./remote_txn_manager/boki_call

.PHONY: remote_txn_mngr_boki_debug
remote_txn_mngr_boki_debug:
	mkdir -p ./bin
	GO111MODULE=on go build -tags "debug" -o bin/remote_txn_mngr_boki_debug ./remote_txn_manager/boki_call

.PHONY: chkptmngr_server
chkptmngr_server:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/chkptmngr_server ./benchmark/common/chkpt_manager/grpc

.PHONY: chkptmngr_server_debug
chkptmngr_server_debug:
	mkdir -p ./bin
	GO111MODULE=on go build -tags "debug" -o bin/chkptmngr_server_debug ./benchmark/common/chkpt_manager/grpc


.PHONY: gen_proto
gen_proto:
	protoc --go_out=. --go_opt=paths=source_relative --proto_path=. \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./pkg/checkpt/chkptmngr_rpc.proto
	protoc --go_out=. --go_opt=paths=source_relative --proto_path=. \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./pkg/commtypes/producer_state.proto
	protoc-go-inject-tag -input=./pkg/commtypes/producer_state.pb.go
	protoc --go_out=. --go_opt=paths=source_relative --proto_path=. \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./pkg/txn_data/txn_meta.proto
	protoc-go-inject-tag -input=./pkg/txn_data/txn_meta.pb.go
	protoc --go_out=. --go_opt=paths=source_relative --proto_path=. \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./pkg/transaction/remote_txn_rpc/remote_txn_rpc.proto
	protoc-go-inject-tag -input=./pkg/transaction/remote_txn_rpc/remote_txn_rpc.pb.go
	go generate ./...

.PHONY: nexmark_scale
nexmark_scale:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/nexmark_scale ./benchmark/nexmark/nexmark_scale

.PHONY: nexmark_genevents_kafka
nexmark_genevents_kafka:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/nexmark_genevents_kafka ./benchmark/nexmark/nexmark_genevents_kafka

.PHONY: nexmark_kafka_test_src
nexmark_kafka_test_src:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/nexmark_kafka_test_src ./benchmark/nexmark/nexmark_kafka_test_src

.PHONY: tests_debug
tests_debug:
	mkdir -p ./bin
	GO111MODULE=on go build -tags debug -o bin/tests_handler_debug ./benchmark/tests/tests_handler

.PHONY: kafka_produce_bench
kafka_produce_bench:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/kafka_produce_bench ./benchmark/lat_tp/kafka_produce

.PHONY: kafka_consume_bench
kafka_consume_bench:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/kafka_consume_bench ./benchmark/lat_tp/kafka_consume

.PHONY: kafka_tran_process
kafka_tran_process:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/kafka_tran_process ./benchmark/protocol_lat/kafka_transaction_process

.PHONY: lat_tp_handler
lat_tp_handler:
	mkdir -p ./bin
	GO111MODULE=on go build -tags stats -o bin/lat_tp_handler ./benchmark/lat_tp/lat_tp_handler

.PHONY: lat_tp_handler_debug
lat_tp_handler_debug:
	mkdir -p ./bin
	GO111MODULE=on go build -tags "stats,debug" -o bin/lat_tp_handler_debug ./benchmark/lat_tp/lat_tp_handler

.PHONY: sharedlog_bench_client
sharedlog_bench_client:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/sharedlog_bench_client ./benchmark/lat_tp/sharedlog_bench_client

.PHONY: sharedlog_protocol_lat
sharedlog_protocol_lat:
	mkdir -p ./bin
	GO111MODULE=on go build -o bin/sharedlog_protocol_lat ./benchmark/lat_tp/sharedlog_protocol_lat

.PHONY: download-book
download-book:
	mkdir -p data
	wget -O data/books.dat https://raw.githubusercontent.com/mayconbordin/storm-applications/master/data/books.dat

.PHONY: java
java:
	cd ./benchmark/lat_tp/kafka_consume_java && gradle uberJar && cd -
	cd ./benchmark/lat_tp/kafka_produce_java && gradle uberJar && cd -
	cd ./benchmark/protocol_lat_java && gradle uberJar && cd -
	
