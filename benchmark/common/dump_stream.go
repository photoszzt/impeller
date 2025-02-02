package common

import (
	"context"
	"fmt"
	"os"
	"path"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/sharedlog_stream"

	"golang.org/x/xerrors"
)

type DumpOutputStreamConfig struct {
	MsgSerde      commtypes.MessageSerde
	OutputDir     string
	TopicName     string
	BufMaxSize    uint32
	SerdeFormat   commtypes.SerdeFormat
	NumPartitions uint8
}

func DumpOutputStream(ctx context.Context, args DumpOutputStreamConfig) error {
	log, err := sharedlog_stream.NewShardedSharedLogStream(args.TopicName, args.NumPartitions,
		args.SerdeFormat, args.BufMaxSize)
	if err != nil {
		return err
	}
	epochMarkerSerde, err := commtypes.GetEpochMarkerSerdeG(args.SerdeFormat)
	if err != nil {
		return err
	}
	payloadArrSerde := sharedlog_stream.DEFAULT_PAYLOAD_ARR_SERDE
	err = os.MkdirAll(args.OutputDir, 0755)
	if err != nil {
		return err
	}
	for i := uint8(0); i < args.NumPartitions; i++ {
		outFilePath := path.Join(args.OutputDir, fmt.Sprintf("%s-%d.txt", args.TopicName, i))
		outFile, err := os.Create(outFilePath)
		if err != nil {
			return err
		}
		off, err := log.SyncToRecent(ctx, i, commtypes.EmptyProducerId)
		if err != nil {
			return err
		}
		for {
			// fmt.Fprintf(os.Stderr, "before read next\n")
			rawMsg, err := log.ReadNext(ctx, i)
			if err != nil {
				if xerrors.Is(err, common_errors.ErrStreamEmpty) && log.GetCuror(i) >= off {
					break
				}
				return err
			}
			if rawMsg.IsControl {
				epochMark, err := epochMarkerSerde.Decode(rawMsg.Payload)
				if err != nil {
					return err
				}
				outStr := fmt.Sprintf("%+v, logSeq: %#x, prodId: %s, auxData: %v\n",
					epochMark, rawMsg.LogSeqNum, rawMsg.ProdId.String(), rawMsg.AuxData)
				// fmt.Fprint(os.Stderr, outStr)
				writted, err := outFile.WriteString(outStr)
				if err != nil {
					return err
				}
				if writted != len(outStr) {
					panic("written is smaller than expected")
				}
			} else if !rawMsg.IsSyncToRecent {
				msgAndSeq, err := commtypes.DecodeRawMsg(rawMsg, args.MsgSerde, payloadArrSerde)
				if err != nil {
					return err
				}
				if msgAndSeq.MsgArr != nil {
					for _, msg := range msgAndSeq.MsgArr {
						err = outputMsg(msg, rawMsg.LogSeqNum, rawMsg.ProdId, outFile)
						if err != nil {
							return err
						}
					}
				} else {
					err = outputMsg(msgAndSeq.Msg, rawMsg.LogSeqNum, rawMsg.ProdId, outFile)
					if err != nil {
						return err
					}
				}
			}
		}
		if err := outFile.Close(); err != nil {
			print(err)
		}
	}
	return nil
}

func outputMsg(msg commtypes.Message, logSeqNum uint64, prodId commtypes.ProducerId, outFile *os.File) error {
	outStr := fmt.Sprintf("%v : %v, ts %#x, logSeq %#x, prodId %s\n",
		msg.Key, msg.Value, msg.Timestamp, logSeqNum, prodId.String())
	// fmt.Fprint(os.Stderr, outStr)
	writted, err := outFile.WriteString(outStr)
	if err != nil {
		return err
	}
	if writted != len(outStr) {
		panic("written is smaller than expected")
	}
	return nil
}
