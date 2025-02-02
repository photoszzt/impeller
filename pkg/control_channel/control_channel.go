package control_channel

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/data_structure"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/snapshot_store"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/store_restore"
	"time"

	// "sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/txn_data"

	"golang.org/x/sync/errgroup"
)

const (
	CONTROL_LOG_TOPIC_NAME = "__control_log"
)

type keyMeta struct {
	hash      uint64
	substream uint8
}

type ControlChannelManager struct {
	payloadArrSerde    commtypes.SerdeG[commtypes.PayloadArr]
	ctrlMetaSerde      commtypes.SerdeG[txn_data.ControlMetadata]
	uint16SerdeG       commtypes.SerdeG[uint16]
	payloadArrSerdeG   commtypes.SerdeG[commtypes.PayloadArr]
	controlOutput      chan ControlChannelResult
	controlLogForRead  *sharedlog_stream.ShardedSharedLogStream
	controlLogForWrite *sharedlog_stream.ShardedSharedLogStream
	controlQuit        chan struct{}
	funcName           string
	ctrlMetaTag        uint64
	ctrlLogTag         uint64
	currentEpoch       uint16
	instanceID         uint8
}

func (cm *ControlChannelManager) CurrentEpoch() uint16 {
	return cm.currentEpoch
}

func NewControlChannelManager(
	app_id string,
	serdeFormat commtypes.SerdeFormat,
	bufMaxSize uint32,
	epoch uint16,
	instanceID uint8,
) (*ControlChannelManager, error) {
	logForRead, err := sharedlog_stream.NewShardedSharedLogStream(CONTROL_LOG_TOPIC_NAME+"_"+app_id, 1,
		serdeFormat, bufMaxSize)
	if err != nil {
		return nil, err
	}
	logForWrite, err := sharedlog_stream.NewShardedSharedLogStream(
		CONTROL_LOG_TOPIC_NAME+"_"+app_id, 1, serdeFormat, bufMaxSize)
	if err != nil {
		return nil, err
	}
	payloadSerde, err := commtypes.GetPayloadArrSerdeG(serdeFormat)
	if err != nil {
		return nil, err
	}
	cm := &ControlChannelManager{
		uint16SerdeG:       commtypes.Uint16SerdeG{},
		payloadArrSerdeG:   payloadSerde,
		controlLogForRead:  logForRead,
		controlLogForWrite: logForWrite,
		currentEpoch:       epoch,
		// keyMappings:        skipmap.NewString[*skipmap.FuncMap[[]byte, keyMeta]](),
		funcName:        app_id,
		payloadArrSerde: sharedlog_stream.DEFAULT_PAYLOAD_ARR_SERDEG,
		instanceID:      instanceID,
		ctrlMetaTag:     txn_data.CtrlMetaTag(logForRead.TopicNameHash(), 0),
		ctrlLogTag:      sharedlog_stream.NameHashWithPartition(logForRead.TopicNameHash(), 0),
	}
	// fmt.Fprintf(os.Stderr, "ctrllog name: %s, tag: 0x%x\n", logForRead.TopicName(), cm.ctrlMetaTag)
	ctrlMetaSerde, err := txn_data.GetControlMetadataSerdeG(serdeFormat)
	if err != nil {
		return nil, err
	}
	cm.ctrlMetaSerde = ctrlMetaSerde
	return cm, nil
}

type restoreWork struct {
	topic  string
	isKV   bool
	parNum uint8
}

func (cmm *ControlChannelManager) loadAndDecodeSnapshot(
	ctx context.Context, topic string,
	rs *snapshot_store.RedisSnapshotStore,
	auxData []byte, metaSeqNum uint64, instanceId uint8,
) ([][]byte, error) {
	ret, err := cmm.uint16SerdeG.Decode(auxData)
	if err != nil {
		return nil, fmt.Errorf("[ERR] Decode: %v", err)
	}
	if ret == 1 {
		snapArr, err := rs.GetSnapshot(ctx, topic, metaSeqNum, instanceId)
		if err != nil {
			return nil, fmt.Errorf("[ERR] RedisGetSnapshot: topic=%s, seq=%#x, err=%v",
				topic, metaSeqNum, err)
		}
		if len(snapArr) > 0 {
			payloadArr, err := cmm.payloadArrSerde.Decode(snapArr)
			if err != nil {
				return nil, fmt.Errorf("[ERR] Decode snap: %v", err)
			}
			return payloadArr.Payloads, nil
		}
	} else {
		fmt.Fprintf(os.Stderr, "no snapshot for %s, metaSeqNum %#x\n", topic, metaSeqNum)
	}
	return nil, nil
}

func (cmm *ControlChannelManager) loadSnapshotToKV(
	ctx context.Context,
	work restoreWork,
	kvc store.KeyValueStoreOpWithChangelog,
	rs *snapshot_store.RedisSnapshotStore,
) error {
	auxData, metaSeqNum, err := kvc.FindLastEpochMetaWithAuxData(ctx, work.parNum)
	if err != nil {
		return err
	}
	// no snapshot
	if auxData == nil {
		return nil
	}
	payloads, err := cmm.loadAndDecodeSnapshot(ctx, work.topic, rs, auxData, metaSeqNum, kvc.GetInstanceId())
	if err != nil {
		return err
	}
	if len(payloads) > 0 {
		err = kvc.RestoreFromSnapshot(payloads)
		if err != nil {
			return err
		}
		kvc.Stream().SetCursor(metaSeqNum+1, kvc.SubstreamNum())
	}
	return nil
}

func (cmm *ControlChannelManager) loadSnapshotToWinStore(
	ctx context.Context,
	work restoreWork,
	wsc store.WindowStoreOpWithChangelog,
	rs *snapshot_store.RedisSnapshotStore,
) error {
	auxData, metaSeqNum, err := wsc.FindLastEpochMetaWithAuxData(ctx, work.parNum)
	if err != nil {
		return err
	}
	// no snapshot
	if auxData == nil {
		return nil
	}
	payloads, err := cmm.loadAndDecodeSnapshot(ctx, work.topic, rs, auxData, metaSeqNum, wsc.GetInstanceId())
	if err != nil {
		return err
	}
	if len(payloads) > 0 {
		err = wsc.RestoreFromSnapshot(ctx, payloads)
		if err != nil {
			return err
		}
		wsc.Stream().SetCursor(metaSeqNum+1, wsc.SubstreamNum())
	}
	return nil
}

func (cmm *ControlChannelManager) restoreFunc(
	bgCtx context.Context, createSnapshot bool, work restoreWork,
	kvchangelog map[string]store.KeyValueStoreOpWithChangelog,
	wschangelog map[string]store.WindowStoreOpWithChangelog,
	rs *snapshot_store.RedisSnapshotStore,
) error {
	if work.isKV {
		kvc := kvchangelog[work.topic]
		if createSnapshot {
			err := cmm.loadSnapshotToKV(bgCtx, work, kvc, rs)
			if err != nil {
				return fmt.Errorf("cmmLoadSnapshotToKV: %v", err)
			}
		}
		err := store_restore.RestoreChangelogKVStateStore(bgCtx, kvc, work.parNum)
		if err != nil {
			return fmt.Errorf("cmmRestoreChangelogKVStateStore: %v", err)
		}
	} else {
		wsc := wschangelog[work.topic]
		if createSnapshot {
			err := cmm.loadSnapshotToWinStore(bgCtx, work, wsc, rs)
			if err != nil {
				return fmt.Errorf("cmmLoadSnapshotToWinStore: %v", err)
			}
		}
		err := store_restore.RestoreChangelogWindowStateStore(bgCtx, wsc, work.parNum)
		if err != nil {
			return fmt.Errorf("cmmRestoreChangelogWindowStateStore: %v", err)
		}
	}
	return nil
}

func (cmm *ControlChannelManager) RestoreMappingAndWaitForPrevTask(
	ctx context.Context, funcName string,
	createSnapshot bool,
	serdeFormat commtypes.SerdeFormat,
	kvchangelog map[string]store.KeyValueStoreOpWithChangelog,
	wschangelog map[string]store.WindowStoreOpWithChangelog,
	rs *snapshot_store.RedisSnapshotStore,
) error {
	extraParToRestoreKV := make(map[string]data_structure.Uint8Set)
	extraParToRestoreWS := make(map[string]data_structure.Uint8Set)
	bgGrp, bgCtx := errgroup.WithContext(ctx)
	prevInstances := make(map[string]data_structure.Uint8Set)
	for {
		rawMsg, err := cmm.controlLogForRead.ReadNext(ctx, 0)
		if err != nil {
			if common_errors.IsStreamEmptyError(err) {
				time.Sleep(time.Duration(100) * time.Millisecond)
				continue
			}
			bgErr := bgGrp.Wait()
			if bgErr != nil {
				return fmt.Errorf("wait bg1: %v", bgErr)
			}
			return fmt.Errorf("ReadNext CtrlLog: %v", err)
		}
		ctrlMeta, err := cmm.ctrlMetaSerde.Decode(rawMsg.Payload)
		if err != nil {
			bgErr := bgGrp.Wait()
			if bgErr != nil {
				return fmt.Errorf("wait bg2: %v", bgErr)
			}
			return fmt.Errorf("ctrlMetaSerde err: %v", err)
		}
		if ctrlMeta.FinishedPrevTask != "" {
			fmt.Fprintf(os.Stderr, "[%d] finished prev task %s, funcName %s, meta epoch %d, current epoch %d, cmm instance %d, remain instances: %v\n",
				ctrlMeta.InstanceId, ctrlMeta.FinishedPrevTask, funcName, ctrlMeta.Epoch, cmm.currentEpoch, cmm.instanceID, prevInstances[funcName])
			if ctrlMeta.FinishedPrevTask == funcName && ctrlMeta.Epoch+1 == cmm.currentEpoch {
				ins := prevInstances[funcName]
				ins.Remove(ctrlMeta.InstanceId)
				prevInstances[funcName] = ins
				if len(ins) == 0 {
					// fmt.Fprintf(os.Stderr, "waiting bg to finish\n")
					err = bgGrp.Wait()
					if err != nil {
						return fmt.Errorf("wait bg3: %v", err)
					}
					return nil
				}
			}
		} else if len(ctrlMeta.Config) != 0 {
			// fmt.Fprintf(os.Stderr, "[%d] scale config: %v, epoch: %d, input epoch: %d\n", cmm.instanceID,
			// 	ctrlMeta.Config, ctrlMeta.Epoch, cmm.currentEpoch)
			if ctrlMeta.Epoch+1 == cmm.currentEpoch {
				for tp, numins := range ctrlMeta.Config {
					ins, ok := prevInstances[tp]
					if !ok {
						ins = data_structure.NewUint8Set()
					}
					for i := uint8(0); i < numins; i++ {
						ins.Add(i)
					}
					prevInstances[tp] = ins
				}
				fmt.Fprintf(os.Stderr, "[%d] prevInstances: %v\n", cmm.instanceID, prevInstances)
			}
		} else if len(ctrlMeta.KeyMaps) != 0 {
			// cmm.updateKeyMapping(&ctrlMeta)
			for tp, kms := range ctrlMeta.KeyMaps {
				kvc, hasKVC := kvchangelog[tp]
				wsc, hasWSC := wschangelog[tp]
				var pars data_structure.Uint8Set
				var parNum uint8
				var ok bool
				var isKV bool
				if hasKVC {
					pars, ok = extraParToRestoreKV[tp]
					parNum = kvc.Stream().NumPartition()
					isKV = true
				} else if hasWSC {
					pars, ok = extraParToRestoreWS[tp]
					parNum = wsc.Stream().NumPartition()
					isKV = false
				}
				if hasKVC || hasWSC {
					if !ok {
						pars = data_structure.NewUint8Set()
					}
					for _, km := range kms {
						// compute the new key assignment
						par := uint8(km.Hash % uint64(parNum))
						// if this key is managed by this task and it was managed by another task in the previous configuration
						if par == cmm.instanceID && km.SubstreamId != cmm.instanceID {
							if !pars.Has(km.SubstreamId) {
								fmt.Fprintf(os.Stderr, "[%d] restore par %d\n", cmm.instanceID, km.SubstreamId)
								w := restoreWork{topic: tp, isKV: isKV, parNum: km.SubstreamId}
								bgGrp.Go(func() error {
									return cmm.restoreFunc(bgCtx, createSnapshot, w, kvchangelog, wschangelog, rs)
								})
								pars.Add(km.SubstreamId)
							}
						}
					}
					if hasKVC {
						extraParToRestoreKV[tp] = pars
					} else {
						extraParToRestoreWS[tp] = pars
					}
				}
			}
		}
	}
}

func (cmm *ControlChannelManager) appendToControlLog(ctx context.Context, cm txn_data.ControlMetadata) error {
	msg_encoded, b, err := cmm.ctrlMetaSerde.Encode(cm)
	defer func() {
		if cmm.ctrlMetaSerde.UsedBufferPool() && b != nil {
			*b = msg_encoded
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return err
	}
	// var off uint64
	_, err = cmm.controlLogForWrite.PushWithTag(ctx, msg_encoded, 0, []uint64{cmm.ctrlMetaTag, cmm.ctrlLogTag}, nil,
		sharedlog_stream.SingleDataRecordMeta, commtypes.EmptyProducerId)
	// debug.Fprintf(os.Stderr, "appendToControlLog: cm %+v, tag 0x%x, off 0x%x\n", cm, cmm.ctrlMetaTag, off)
	return err
}

func (cmm *ControlChannelManager) OutputKeyMapping(ctx context.Context, kms map[string][]txn_data.KeyMaping) error {
	cm := txn_data.ControlMetadata{
		KeyMaps:    kms,
		InstanceId: cmm.instanceID,
		Epoch:      cmm.currentEpoch,
	}
	return cmm.appendToControlLog(ctx, cm)
}

func (cmm *ControlChannelManager) AppendRescaleConfig(ctx context.Context,
	config map[string]uint8,
) error {
	cmm.currentEpoch += 1
	cm := txn_data.ControlMetadata{
		Config: config,
		Epoch:  cmm.currentEpoch,
	}
	err := cmm.appendToControlLog(ctx, cm)
	return err
}

func (cmm *ControlChannelManager) RecordPrevInstanceFinish(
	ctx context.Context,
	funcName string,
	instanceID uint8,
	epoch uint16,
) error {
	// debug.Fprintf(os.Stderr, "%s(%d) epoch %d finished\n", funcName, instanceID, epoch)
	cm := txn_data.ControlMetadata{
		FinishedPrevTask: funcName,
		InstanceId:       instanceID,
		Epoch:            epoch,
	}
	err := cmm.appendToControlLog(ctx, cm)
	return err
}

func (cmm *ControlChannelManager) StartMonitorControlChannel(ctx context.Context) {
	cmm.controlQuit = make(chan struct{})
	cmm.controlOutput = make(chan ControlChannelResult, 10)
	go cmm.monitorControlChannel(ctx, cmm.controlQuit, cmm.controlOutput)
}

func (cmm *ControlChannelManager) OutputChan() chan ControlChannelResult {
	return cmm.controlOutput
}

func (cmm *ControlChannelManager) SendQuit() {
	cmm.controlQuit <- struct{}{}
}

func (cmm *ControlChannelManager) monitorControlChannel(
	ctx context.Context,
	quit <-chan struct{},
	output chan<- ControlChannelResult,
) {
	// fmt.Fprintf(os.Stderr, "start monitoring control channel from %s 0x%x with tag 0x%x\n",
	// 	cmm.controlLogForRead.TopicName(), cmm.controlLogForRead.GetCuror(0), cmm.ctrlMetaTag)
	for {
		select {
		case <-quit:
			return
		default:
		}

		rawMsg, err := cmm.controlLogForRead.ReadNextWithTag(ctx, 0, cmm.ctrlMetaTag)
		if err != nil {
			if common_errors.IsStreamEmptyError(err) {
				time.Sleep(time.Duration(1) * time.Millisecond)
				continue
			}
			output <- ControlChannelErr(err)
			break
		}
		// fmt.Fprintf(os.Stderr, "read control message from 0x%x\n", rawMsg.LogSeqNum)
		if !rawMsg.IsPayloadArr {
			ctrlMeta, err := cmm.ctrlMetaSerde.Decode(rawMsg.Payload)
			if err != nil {
				output <- ControlChannelErr(err)
				break
			}
			// fmt.Fprintf(os.Stderr, "monitorControlChannel: %+v\n", ctrlMeta)
			output <- ControlChannelVal(&ctrlMeta)
		}
	}
}
