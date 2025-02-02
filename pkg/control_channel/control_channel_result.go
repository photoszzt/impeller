package control_channel

import "sharedlog-stream/pkg/txn_data"

type ControlChannelResult struct {
	err  error
	meta *txn_data.ControlMetadata
}

func ControlChannelErr(err error) ControlChannelResult {
	return ControlChannelResult{
		err: err,
	}
}

func ControlChannelVal(meta *txn_data.ControlMetadata) ControlChannelResult {
	return ControlChannelResult{
		meta: meta,
		err:  nil,
	}
}

func (r *ControlChannelResult) Valid() bool {
	return r.err == nil
}

func (r *ControlChannelResult) Value() *txn_data.ControlMetadata {
	return r.meta
}

func (r *ControlChannelResult) Err() error {
	return r.err
}
