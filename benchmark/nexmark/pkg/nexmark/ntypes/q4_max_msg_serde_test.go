package ntypes

import (
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"testing"
)

func msgEncodeDecode[K, V comparable](t *testing.T, msg commtypes.MessageG[K, V],
	msgSerde commtypes.MessageGSerdeG[K, V]) {

	var msgSerSerde commtypes.SerdeG[commtypes.MessageSerialized]
	var payloadArrSerde commtypes.SerdeG[commtypes.PayloadArr]
	msgSerSerde = commtypes.MessageSerializedMsgpSerdeG{}
	payloadArrSerde = commtypes.PayloadArrMsgpSerdeG{}
	bytes, b, err := msgSerde.Encode(msg)
	if err != nil {
		t.Fatal(err)
	}
	payloadArr := commtypes.PayloadArr{
		Payloads: [][]byte{bytes},
	}
	payloads, b_payload, err := payloadArrSerde.Encode(payloadArr)
	if err != nil {
		t.Fatal(err)
	}

	ret_payload_arr, err := payloadArrSerde.Decode(payloads)
	if err != nil {
		t.Fatal(err)
	}
	ret_msgSer, err := msgSerSerde.Decode(ret_payload_arr.Payloads[0])
	if err != nil {
		t.Fatal(err)
	}
	ret_msg, err := commtypes.MsgSerToMsgG(&ret_msgSer, msgSerde)
	if err != nil {
		t.Fatal(err)
	}
	ret_key := ret_msg.Key.Unwrap()
	ret_val := ret_msg.Value.Unwrap()
	key := msg.Key.Unwrap()
	val := msg.Value.Unwrap()
	if ret_key != key || ret_val != val {
		t.Fatalf("expected: %v, got %v\n", msg, ret_msg)
	}
	*b_payload = payloads
	commtypes.PushBuffer(b_payload)
	*b = bytes
	commtypes.PushBuffer(b)
}

func TestQ4MaxMsgSerde(t *testing.T) {
	var abSerde commtypes.SerdeG[*AuctionBid]
	var aicSerde commtypes.SerdeG[AuctionIdCategory]
	abSerde = AuctionBidMsgpSerdeG{}
	aicSerde = AuctionIdCategoryMsgpSerdeG{}

	msgSerde, err := commtypes.GetMsgGSerdeG(commtypes.MSGP, aicSerde, abSerde)
	if err != nil {
		t.Fatal(err)
	}
	msg := commtypes.MessageG[AuctionIdCategory, *AuctionBid]{
		Value: optional.Some(&AuctionBid{
			BidDateTime: 1,
			AucDateTime: 2,
			AucExpires:  5,
			BidPrice:    3,
			AucCategory: 100,
			AucSeller:   2,
		}),
		Key: optional.Some(AuctionIdCategory{
			AucId:    1,
			Category: 3,
		}),
	}
	msgEncodeDecode(t, msg, msgSerde)
}
