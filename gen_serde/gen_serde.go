package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"log"
	"os"
	"text/template"
)

type Variant struct {
	// Package is the package name.
	PackageName string

	// Name is the variant name: should be unique among variants.
	TypeName string

	// Path is the file path into which the generator will emit the code for this
	// variant.
	Path string

	ExtraImports string

	CommtypesPrefix string
}

func generate(v *Variant, code string) {
	// Parse templateCode anew for each variant because Parse requires Funcs to be
	// registered, and it helps type-check the funcs.
	tmpl, err := template.New("gen").Parse(code)
	if err != nil {
		log.Fatal("template Parse:", err)
	}

	var out bytes.Buffer
	err = tmpl.Execute(&out, v)
	if err != nil {
		log.Fatal("template Execute:", err)
	}

	err = os.WriteFile(v.Path, out.Bytes(), 0644)
	if err != nil {
		log.Fatal("os.WriteFile:", err)
	}

	formatted, err := format.Source(out.Bytes())
	if err != nil {
		println(out.String())
		log.Fatal("format:", err)
	}

	if err := os.WriteFile(v.Path, formatted, 0644); err != nil {
		log.Fatal("WriteFile:", err)
	}
}

func default_variant(fname, typeName, dirpath, packageName string, inCommtypes bool) *Variant {
	v := &Variant{
		PackageName: packageName,
		TypeName:    typeName,
		Path:        fmt.Sprintf("%s/%s_gen_serde.go", dirpath, fname),
	}
	if inCommtypes {
		v.ExtraImports = ""
		v.CommtypesPrefix = ""
	} else {
		v.ExtraImports = "\"sharedlog-stream/pkg/commtypes\""
		v.CommtypesPrefix = "commtypes."
	}
	return v
}

func gen_serde_test(fname, typeName, dirpath, packageName string, inCommtypes bool) {
	v := default_variant(fname, typeName, dirpath, packageName, inCommtypes)
	v.Path = fmt.Sprintf("%s/%s_gen_serde_test.go", dirpath, fname)
	generate(v, serde_test)
}

func gen_serde_ptr_test(fname, typeName, dirpath, packageName string, inCommtypes bool) {
	v := default_variant(fname, typeName, dirpath, packageName, inCommtypes)
	v.Path = fmt.Sprintf("%s/%s_gen_serde_ptr_test.go", dirpath, fname)
	generate(v, serde_ptr_test)
}

func gen_serde(fname, typeName, dirpath, packageName string, inCommtypes bool) {
	v := default_variant(fname, typeName, dirpath, packageName, inCommtypes)
	generate(v, serde)
	v.Path = fmt.Sprintf("%s/%s_gen_serdeG.go", dirpath, fname)
	generate(v, serdeG)
}

func gen_serde_ptr(fname, typeName, dirpath, packageName string, inCommtypes bool) {
	v := default_variant(fname, typeName, dirpath, packageName, inCommtypes)
	generate(v, serde_ptr)
	v.Path = fmt.Sprintf("%s/%s_gen_serdeG.go", dirpath, fname)
	generate(v, serdeG_ptr)
}

func main() {
	ntypes_path := "../benchmark/nexmark/pkg/nexmark/ntypes/"
	gen_serde("start_end_time", "StartEndTime", ntypes_path, "ntypes", false)
	gen_serde("sum_and_count", "SumAndCount", ntypes_path, "ntypes", false)
	gen_serde("price_time_list", "PriceTimeList", ntypes_path, "ntypes", false)
	gen_serde("price_time", "PriceTime", ntypes_path, "ntypes", false)
	gen_serde("person_time", "PersonTime", ntypes_path, "ntypes", false)
	gen_serde("name_city_state_id", "NameCityStateId", ntypes_path, "ntypes", false)
	gen_serde("bid_price", "BidPrice", ntypes_path, "ntypes", false)
	gen_serde("bid_and_max", "BidAndMax", ntypes_path, "ntypes", false)
	gen_serde("auction_id_seller", "AuctionIdSeller", ntypes_path, "ntypes", false)
	gen_serde("auction_id_count", "AuctionIdCount", ntypes_path, "ntypes", false)
	gen_serde("auction_id_cnt_max", "AuctionIdCntMax", ntypes_path, "ntypes", false)
	gen_serde("auction_id_category", "AuctionIdCategory", ntypes_path, "ntypes", false)
	gen_serde_ptr("event", "Event", ntypes_path, "ntypes", false)
	gen_serde_ptr("auction_bid", "AuctionBid", ntypes_path, "ntypes", false)

	gen_serde_test("start_end_time", "StartEndTime", ntypes_path, "ntypes", false)
	gen_serde_test("sum_and_count", "SumAndCount", ntypes_path, "ntypes", false)
	gen_serde_test("price_time", "PriceTime", ntypes_path, "ntypes", false)
	gen_serde_test("price_time_list", "PriceTimeList", ntypes_path, "ntypes", false)
	gen_serde_test("person_time", "PersonTime", ntypes_path, "ntypes", false)
	gen_serde_test("name_city_state_id", "NameCityStateId", ntypes_path, "ntypes", false)
	gen_serde_test("bid_price", "BidPrice", ntypes_path, "ntypes", false)
	gen_serde_test("bid_and_max", "BidAndMax", ntypes_path, "ntypes", false)
	gen_serde_test("auction_id_seller", "AuctionIdSeller", ntypes_path, "ntypes", false)
	gen_serde_test("auction_id_count", "AuctionIdCount", ntypes_path, "ntypes", false)
	gen_serde_test("auction_id_cnt_max", "AuctionIdCntMax", ntypes_path, "ntypes", false)
	gen_serde_test("auction_id_category", "AuctionIdCategory", ntypes_path, "ntypes", false)
	gen_serde_ptr_test("event", "Event", ntypes_path, "ntypes", false)
	gen_serde_ptr_test("auction_bid", "AuctionBid", ntypes_path, "ntypes", false)

	commtypes_path := "../pkg/commtypes/"
	gen_serde("checkpoint", "Checkpoint", commtypes_path, "commtypes", true)
	gen_serde("epoch_meta", "EpochMarker", commtypes_path, "commtypes", true)
	gen_serde("payload_arr", "PayloadArr", commtypes_path, "commtypes", true)
	gen_serde("table_snapshots", "TableSnapshots", commtypes_path, "commtypes", true)
	gen_serde("offset_marker", "OffsetMarker", commtypes_path, "commtypes", true)
	gen_serde("message_serialized", "MessageSerialized", commtypes_path, "commtypes", true)
	gen_serde("time_window", "TimeWindow", commtypes_path, "commtypes", true)

	gen_serde_test("checkpoint", "Checkpoint", commtypes_path, "commtypes", true)
	gen_serde_test("epoch_meta", "EpochMarker", commtypes_path, "commtypes", true)
	gen_serde_test("payload_arr", "PayloadArr", commtypes_path, "commtypes", true)
	gen_serde_test("table_snapshots", "TableSnapshots", commtypes_path, "commtypes", true)
	gen_serde_test("offset_marker", "OffsetMarker", commtypes_path, "commtypes", true)
	gen_serde_test("message_serialized", "MessageSerialized", commtypes_path, "commtypes", true)
	gen_serde_test("time_window", "TimeWindow", commtypes_path, "commtypes", true)

	txn_path := "../pkg/txn_data/"
	gen_serde("control_meta", "ControlMetadata", txn_path, "txn_data", false)
	gen_serde("offset_record", "OffsetRecord", txn_path, "txn_data", false)
	gen_serde_ptr("topic_partition", "TopicPartition", txn_path, "txn_data", false)
	gen_serde("txn_metadata", "TxnMetadata", txn_path, "txn_data", false)

	gen_serde_test("control_meta", "ControlMetadata", txn_path, "txn_data", false)
	gen_serde_test("offset_record", "OffsetRecord", txn_path, "txn_data", false)
	gen_serde_ptr_test("topic_partition", "TopicPartition", txn_path, "txn_data", false)
	gen_serde_test("txn_metadata", "TxnMetadata", txn_path, "txn_data", false)

	rtxn_rpc_path := "../pkg/transaction/remote_txn_rpc/"
	gen_serde_ptr("rtxn_arg", "RTxnArg", rtxn_rpc_path, "remote_txn_rpc", false)
	gen_serde_ptr("rtxn_reply", "RTxnReply", rtxn_rpc_path, "remote_txn_rpc", false)

	gen_serde_ptr_test("rtxn_arg", "RTxnArg", rtxn_rpc_path, "remote_txn_rpc", false)
	gen_serde_ptr_test("rtxn_reply", "RTxnReply", rtxn_rpc_path, "remote_txn_rpc", false)

	lat_tp_path := "../benchmark/lat_tp/pkg/data_type/"
	gen_serde("payload_ts", "PayloadTs", lat_tp_path, "datatype", false)

	gen_serde_test("payload_ts", "PayloadTs", lat_tp_path, "datatype", false)
}

//go:embed serde.tmpl
var serde string

//go:embed serde_ptr.tmpl
var serde_ptr string

//go:embed serdeG.tmpl
var serdeG string

//go:embed serdeG_ptr.tmpl
var serdeG_ptr string

//go:embed serde_ptr_test.tmpl
var serde_ptr_test string

//go:embed serde_test.tmpl
var serde_test string
