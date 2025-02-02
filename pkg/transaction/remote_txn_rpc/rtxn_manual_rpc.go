package remote_txn_rpc

import (
	"bytes"
	context "context"
	"fmt"
	"net/http"
	commtypes "sharedlog-stream/pkg/commtypes"
	txn_data "sharedlog-stream/pkg/txn_data"

	"github.com/tinylib/msgp/msgp"
	"golang.org/x/xerrors"
)

type RemoteTxnClient interface {
	Init(ctx context.Context, in *InitArg) (*InitReply, error)
	AppendTpPar(ctx context.Context, in *txn_data.TxnMetaMsg) error
	AbortTxn(ctx context.Context, in *txn_data.TxnMetaMsg) error
	CommitTxnAsyncComplete(ctx context.Context, in *txn_data.TxnMetaMsg) (*CommitReply, error)
	AppendConsumedOffset(ctx context.Context, in *ConsumedOffsets) error
}

var RTxnFuncName = "remoteTxnMngr"

type RTxnRpcClient struct {
	client         *http.Client
	funcUrl        string
	nodeConstraint string
	readBuffer     []byte
	serdeFormat    commtypes.SerdeFormat // serdeFormat for log
	serde          commtypes.SerdeG[*RTxnArg]
}

func NewRTxnRpcClient(faas_gateway string, nodeConstraint string, serdeFormat commtypes.SerdeFormat) RTxnRpcClient {
	s, _ := GetRTxnArgSerdeG(commtypes.MSGP)
	return RTxnRpcClient{
		client:         &http.Client{},
		funcUrl:        fmt.Sprintf("http://%s/function/%s", faas_gateway, RTxnFuncName),
		nodeConstraint: nodeConstraint,
		readBuffer:     make([]byte, 0, 1024),
		serdeFormat:    serdeFormat,
		serde:          s,
	}
}

func postRequest[V msgp.Decodable](client *http.Client, url, nodeConstraint string, request []byte, response V) error {
	req, err := http.NewRequest("POST", url, bytes.NewReader(request))
	if err != nil {
		return err
	}
	if nodeConstraint != "" {
		req.Header.Add("X-Faas-Node-Constraint", nodeConstraint)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Non-OK response: %d", resp.StatusCode)
	}
	reader := msgp.NewReader(resp.Body)
	return response.DecodeMsg(reader)
}

func (c *RTxnRpcClient) Init(ctx context.Context, in *InitArg) (*InitReply, error) {
	arg := &RTxnArg{
		RpcType:     Init,
		SerdeFormat: uint8(c.serdeFormat),
		Init:        in,
	}
	ret, b, err := c.serde.Encode(arg)
	defer func() {
		if b != nil {
			*b = ret
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return nil, err
	}
	reply := RTxnReply{}
	err = postRequest(c.client, c.funcUrl, c.nodeConstraint, ret, &reply)
	if err != nil {
		return nil, err
	}
	if reply.Success {
		return reply.InitReply, nil
	} else {
		return nil, xerrors.New(reply.Message)
	}
}

func (c *RTxnRpcClient) AppendTpPar(ctx context.Context, in *txn_data.TxnMetaMsg) error {
	arg := &RTxnArg{
		RpcType:     AppendTpPar,
		SerdeFormat: uint8(c.serdeFormat),
		MetaMsg:     in,
	}
	ret, b, err := c.serde.Encode(arg)
	defer func() {
		if b != nil {
			*b = ret
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return err
	}
	reply := RTxnReply{}
	err = postRequest(c.client, c.funcUrl, c.nodeConstraint, ret, &reply)
	if err != nil {
		return err
	}
	if reply.Success {
		return nil
	} else {
		return xerrors.New(reply.Message)
	}
}

func (c *RTxnRpcClient) AbortTxn(ctx context.Context, in *txn_data.TxnMetaMsg) error {
	arg := &RTxnArg{
		RpcType:     AbortTxn,
		SerdeFormat: uint8(c.serdeFormat),
		MetaMsg:     in,
	}
	ret, b, err := c.serde.Encode(arg)
	defer func() {
		if b != nil {
			*b = ret
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return err
	}
	reply := RTxnReply{}
	err = postRequest(c.client, c.funcUrl, c.nodeConstraint, ret, &reply)
	if err != nil {
		return err
	}
	if reply.Success {
		return nil
	} else {
		return xerrors.New(reply.Message)
	}
}

func (c *RTxnRpcClient) CommitTxnAsyncComplete(ctx context.Context, in *txn_data.TxnMetaMsg) (*CommitReply, error) {
	arg := &RTxnArg{
		RpcType:     CommitTxnAsync,
		SerdeFormat: uint8(c.serdeFormat),
		MetaMsg:     in,
	}
	ret, b, err := c.serde.Encode(arg)
	defer func() {
		if b != nil {
			*b = ret
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return nil, err
	}
	reply := RTxnReply{}
	err = postRequest(c.client, c.funcUrl, c.nodeConstraint, ret, &reply)
	if err != nil {
		return nil, err
	}
	if reply.Success {
		return reply.CommitReply, nil
	} else {
		return nil, xerrors.New(reply.Message)
	}
}

func (c *RTxnRpcClient) AppendConsumedOffset(ctx context.Context, in *ConsumedOffsets) error {
	arg := &RTxnArg{
		RpcType:     AppendConsumedOff,
		SerdeFormat: uint8(c.serdeFormat),
		ConsumedOff: in,
	}
	ret, b, err := c.serde.Encode(arg)
	defer func() {
		if b != nil {
			*b = ret
			commtypes.PushBuffer(&ret)
		}
	}()
	if err != nil {
		return err
	}
	reply := RTxnReply{}
	err = postRequest(c.client, c.funcUrl, c.nodeConstraint, ret, &reply)
	if err != nil {
		return err
	}
	if reply.Success {
		return nil
	} else {
		return xerrors.New(reply.Message)
	}
}
