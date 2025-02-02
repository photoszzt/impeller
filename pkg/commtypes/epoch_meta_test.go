package commtypes

import "testing"

func TestSerdeEpochMarker2(t *testing.T) {
	v := EpochMarker{
		ConSeqNums: map[string]uint64{
			"a": 1,
			"b": 2,
		},
		OutputRanges: map[string][]ProduceRange{
			"a": {
				{Start: 1, SubStreamNum: 0},
				{Start: 3, SubStreamNum: 1},
			},
			"b": {
				{Start: 5, SubStreamNum: 2},
				{Start: 7, SubStreamNum: 3},
			},
		},
		StartTime:  1,
		ScaleEpoch: 0,
		ProdIndex:  1,
		Mark:       EPOCH_END,
	}
	jsonSerdeG := EpochMarkerJSONSerdeG{}
	jsonSerde := EpochMarkerJSONSerde{}
	msgSerdeG := EpochMarkerMsgpSerdeG{}
	msgSerde := EpochMarkerMsgpSerde{}
	GenTestEncodeDecode(v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecode(v, t, msgSerdeG, msgSerde)
}
