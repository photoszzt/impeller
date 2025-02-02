package common

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sharedlog-stream/pkg/checkpt"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"strconv"
	"sync"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/rs/zerolog/log"
)

type SrcInvokeConfig struct {
	FinalTpNames    []string
	TopicName       string
	AppId           string
	NodeConstraint  string
	AdditionalBytes int
	ScaleEpoch      uint16
	NumOutPartition uint8
	InstanceID      uint8
	NumSrcInstance  uint8
}

func (c *SrcInvokeConfig) Clone() SrcInvokeConfig {
	return SrcInvokeConfig{
		FinalTpNames:    c.FinalTpNames,
		TopicName:       c.TopicName,
		AppId:           c.AppId,
		NodeConstraint:  c.NodeConstraint,
		AdditionalBytes: c.AdditionalBytes,
		ScaleEpoch:      c.ScaleEpoch,
		NumOutPartition: c.NumOutPartition,
		InstanceID:      c.InstanceID,
		NumSrcInstance:  c.NumSrcInstance,
	}
}

type InvokeFuncParam struct {
	ConfigFile     string
	StatDir        string
	GatewayUrl     string
	WarmupTime     int
	Local          bool
	WaitForEndMark bool
}

type InvokeParams struct {
	SrcInvokeConfig  *SrcInvokeConfig
	CliNodes         []*ClientNode
	InParamsMap      map[string][]*QueryInput
	ConfigScaleInput *ConfigScaleInput
	FinalOutPars     []uint8
}

func ParseInvokeParam(invokeParam InvokeFuncParam, baseQueryInput *QueryInput,
) (
	params *InvokeParams,
	err error,
) {
	byteVal, err := os.ReadFile(invokeParam.ConfigFile)
	if err != nil {
		return nil, err
	}
	jsonParsed, err := gabs.ParseJSON(byteVal)
	if err != nil {
		return nil, err
	}
	funcParam := jsonParsed.S("FuncParam").Children()
	streamParam := jsonParsed.S("StreamParam").ChildrenMap()

	var cliNodes []*ClientNode
	srcInvokeConfig := &SrcInvokeConfig{}
	scaleConfig := make(map[string]uint8)
	inParamsMap := make(map[string][]*QueryInput)
	funcNames := make([]string, 0)
	changelogTmp, hasChangelog := streamParam["changelog"]
	changelog := uint8(0)
	if hasChangelog {
		changelog = uint8(changelogTmp.Data().(float64))
	}
	var finalOutPars []uint8
	for _, child := range funcParam {
		config := child.ChildrenMap()
		fmt.Fprintf(os.Stderr, "config: %+v\n", config)
		ninstance := uint8(config["NumInstance"].Data().(float64))
		funcName := config["funcName"].Data().(string)
		outputTopicNamesTmp := config["OutputTopicName"].Data().([]interface{})
		inputTopicNamesTmp, ok := config["InputTopicNames"]
		_, hasFinal := config["Final"]
		numSrcProducerTmp := config["NumSrcProducer"]
		scaleConfig[funcName] = ninstance
		funcNames = append(funcNames, funcName)
		nodeConstraint := config["NodeConstraint"].Data().(string)
		if invokeParam.Local {
			nodeConstraint = ""
		}

		if !ok {
			// this is source config
			srcInvokeConfig.NumSrcInstance = ninstance
			srcInvokeConfig.TopicName = outputTopicNamesTmp[0].(string)
			srcInvokeConfig.NodeConstraint = nodeConstraint
			srcInvokeConfig.AppId = baseQueryInput.AppId
		} else {
			inputTopicNamesIntf := inputTopicNamesTmp.Data().([]interface{})
			numSrcProducerIntf := numSrcProducerTmp.Data().([]interface{})
			numSrcProducer := make([]uint8, len(numSrcProducerIntf))
			inputTopicNames := make([]string, len(inputTopicNamesIntf))
			outputTopicNames := make([]string, len(outputTopicNamesTmp))
			numOutPartitions := make([]uint8, len(outputTopicNamesTmp))

			for i, v := range inputTopicNamesIntf {
				inputTopicNames[i] = v.(string)
			}
			for i, v := range numSrcProducerIntf {
				numSrcProducer[i] = uint8(v.(float64))
			}
			for i, v := range outputTopicNamesTmp {
				outputTopicNames[i] = v.(string)
				numOutPartitions[i] = uint8(streamParam[outputTopicNames[i]].Data().(float64))
			}
			if hasFinal {
				srcInvokeConfig.FinalTpNames = outputTopicNames
				finalOutPars = numOutPartitions
			}
			nconfig := &ClientNodeConfig{
				FuncName:       funcName,
				GatewayUrl:     invokeParam.GatewayUrl,
				NumInstance:    ninstance,
				NodeConstraint: nodeConstraint,
			}
			inParams := make([]*QueryInput, ninstance)
			for i := uint8(0); i < ninstance; i++ {
				numInSubs := uint8(streamParam[inputTopicNames[0]].Data().(float64))
				baseClone := baseQueryInput.Clone()
				baseClone.InputTopicNames = inputTopicNames
				baseClone.OutputTopicNames = outputTopicNames
				baseClone.NumSubstreamProducer = numSrcProducer
				baseClone.NumInPartition = numInSubs
				baseClone.NumOutPartitions = numOutPartitions
				baseClone.WaitForEndMark = invokeParam.WaitForEndMark
				if hasChangelog {
					baseClone.NumChangelogPartition = changelog
				}
				inParams[i] = &baseClone
			}
			node := NewClientNode(nconfig)
			cliNodes = append(cliNodes, node)
			inParamsMap[funcName] = inParams
		}
	}
	for tp, subs := range streamParam {
		scaleConfig[tp] = uint8(subs.Data().(float64))
	}
	srcInvokeConfig.NumOutPartition = uint8(streamParam[srcInvokeConfig.TopicName].Data().(float64))
	configScaleInput := &ConfigScaleInput{
		Config:      scaleConfig,
		FuncNames:   funcNames,
		SerdeFormat: baseQueryInput.SerdeFormat,
		AppId:       baseQueryInput.AppId,
		BufMaxSize:  baseQueryInput.BufMaxSize,
	}
	return &InvokeParams{
		SrcInvokeConfig:  srcInvokeConfig,
		CliNodes:         cliNodes,
		InParamsMap:      inParamsMap,
		ConfigScaleInput: configScaleInput,
		FinalOutPars:     finalOutPars,
	}, nil
}

func InvokeSrc(wg *sync.WaitGroup, client *http.Client,
	srcInvokeConfig *SrcInvokeConfig,
	invokeSourceFunc InvokeSrcFunc,
	scaleEpoch uint16,
) []FnOutput {
	sourceOutput := make([]FnOutput, srcInvokeConfig.NumSrcInstance)
	for i := uint8(0); i < srcInvokeConfig.NumSrcInstance; i++ {
		wg.Add(1)
		idx := i
		srcInvokeConfigCpy := srcInvokeConfig.Clone()
		srcInvokeConfigCpy.ScaleEpoch = scaleEpoch
		srcInvokeConfigCpy.InstanceID = idx
		go invokeSourceFunc(client, srcInvokeConfigCpy, &sourceOutput[idx], wg, false)
	}
	return sourceOutput
}

func InvokeFunctions(wg *sync.WaitGroup, client *http.Client, cliNodes []*ClientNode,
	inParamsMap map[string][]*QueryInput, scaleEpoch uint16,
) map[string][]FnOutput {
	outputMap := make(map[string][]FnOutput)
	for _, node := range cliNodes {
		outputMap[node.Name()] = make([]FnOutput, len(inParamsMap[node.Name()]))
	}
	for _, node := range cliNodes {
		funcName := node.Name()
		param := inParamsMap[funcName]
		output := outputMap[funcName]
		for j := uint8(0); j < uint8(len(param)); j++ {
			wg.Add(1)
			idx := j
			param[idx].ParNum = idx
			param[idx].Duration = param[0].Duration
			param[idx].ScaleEpoch = scaleEpoch
			go node.Invoke(client, &output[idx], wg, param[idx])
		}
	}
	return outputMap
}

func ParseSrcOutput(sourceOutput []FnOutput, statsDir string) {
	srcNum := make(map[string]uint64)
	srcEndToEnd := float64(0)
	maxDuration := float64(0)
	for idx, out := range sourceOutput {
		if out.Success {
			if maxDuration < sourceOutput[idx].Duration {
				maxDuration = sourceOutput[idx].Duration
			}
			ProcessThroughputLat(fmt.Sprintf("source-%d", idx),
				statsDir, sourceOutput[idx].Latencies, sourceOutput[idx].Counts,
				sourceOutput[idx].EventTs, sourceOutput[idx].Duration, srcNum, &srcEndToEnd)
		} else {
			fmt.Fprintf(os.Stderr, "source-%d failed\n", idx)
		}
	}
	if len(srcNum) != 0 {
		for k, v := range srcNum {
			fmt.Fprintf(os.Stderr, "%s processed %v events, duration: %v, tp: %f\n", k, v, maxDuration, float64(v)/maxDuration)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}
}

func ParseFunctionOutputs(outputMap map[string][]FnOutput, statDir string) {
	for funcName, output := range outputMap {
		num := make(map[string]uint64)
		endToEnd := float64(0)
		maxDuration := float64(0)
		for j := uint8(0); j < uint8(len(output)); j++ {
			if output[j].Success {
				if maxDuration < output[j].Duration {
					maxDuration = output[j].Duration
				}
				ProcessThroughputLat(fmt.Sprintf("%s-%d", funcName, j),
					statDir, output[j].Latencies, output[j].Counts, output[j].EventTs, output[j].Duration,
					num, &endToEnd)
			} else {
				fmt.Fprintf(os.Stderr, "%s-%d failed, msg %s\n", funcName, j, output[j].Message)
			}
		}
		if len(num) != 0 {
			for k, v := range num {
				fmt.Fprintf(os.Stderr, "%s processed %v events, duration: %v\n", k, v, maxDuration)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
	}
}

type InvokeSrcFunc func(client *http.Client, srcInvokeConfig SrcInvokeConfig, response *FnOutput, wg *sync.WaitGroup, warmup bool)

func InvokeRTxnMngr(client *http.Client, input *RTxnMngrInput, faas_gateway string, local bool, rtxnNodeIdStart int) {
	appName := "remoteTxnMngr"
	url := BuildFunctionUrl(faas_gateway, appName)
	fmt.Fprintf(os.Stderr, "rtxnmngr url is %s\n", url)
	constraint := ""
	for i := rtxnNodeIdStart; i < rtxnNodeIdStart+4; i++ {
		if !local {
			constraint = strconv.Itoa(i)
		}
		go func(client *http.Client, url, constraint string) {
			var response FnOutput
			if err := JsonPostRequest(client, url, constraint, input, &response); err != nil {
				log.Error().Msgf("%s request failed: %v", appName, err)
			} else if !response.Success {
				log.Error().Msgf("%s request failed: %s", appName, response.Message)
			}
		}(client, url, constraint)
	}
}

func InvokeChkMngr(wg *sync.WaitGroup, client *http.Client, cmi *ChkptMngrInput,
	faas_gateway string, local bool,
) {
	defer wg.Done()
	var response FnOutput
	appName := "chkptmngr"
	url := BuildFunctionUrl(faas_gateway, appName)
	fmt.Fprintf(os.Stderr, "chkptmngr url is %s\n", url)
	constraint := "1"
	if local {
		constraint = ""
	}
	if err := JsonPostRequest(client, url, constraint, cmi, &response); err != nil {
		log.Error().Msgf("%s request failed: %v", appName, err)
	} else if !response.Success {
		log.Error().Msgf("%s request failed: %s", appName, response.Message)
	}
}

func InvokeRedisSetup(client *http.Client, rsi *RedisSetupInput, faas_gateway string, local bool) {
	var response FnOutput
	appName := "redisSetup"
	url := BuildFunctionUrl(faas_gateway, appName)
	fmt.Fprintf(os.Stderr, "redis_setup url is %s\n", url)
	constraint := "1"
	if local {
		constraint = ""
	}
	if err := JsonPostRequest(client, url, constraint, rsi, &response); err != nil {
		log.Error().Msgf("%s request failed: %v", appName, err)
	} else if !response.Success {
		log.Error().Msgf("%s request failed: %s", appName, response.Message)
	}
}

func Invoke(ctx context.Context, invokeParam InvokeFuncParam,
	baseQueryInput *QueryInput,
	invokeSourceFunc InvokeSrcFunc,
	additionalBytes int,
	fixedMaxDur uint32,
) (params *InvokeParams, err error) {
	params, err = ParseInvokeParam(invokeParam, baseQueryInput)
	if err != nil {
		return nil, err
	}
	params.SrcInvokeConfig.AdditionalBytes = additionalBytes
	fmt.Fprintf(os.Stderr, "srcInvokeConfig: %+v\n", params.SrcInvokeConfig)
	fmt.Fprintf(os.Stderr, "cliNodes: %+v\n", params.CliNodes)
	fmt.Fprintf(os.Stderr, "inParamsMap: %+v\n", params.InParamsMap)
	fmt.Fprintf(os.Stderr, "configScaleInput: %+v\n", params.ConfigScaleInput)

	to := baseQueryInput.Duration + 60
	if to < fixedMaxDur {
		to = fixedMaxDur
	}

	timeout := time.Duration(to) * time.Second
	client := &http.Client{
		Transport: &http.Transport{
			IdleConnTimeout: 30 * time.Second,
		},
		Timeout: timeout,
	}
	params.ConfigScaleInput.Bootstrap = true
	params.ConfigScaleInput.ScaleEpoch = 1
	var scaleResponse FnOutput
	InvokeConfigScale(client, params.ConfigScaleInput, invokeParam.GatewayUrl,
		&scaleResponse, "scale", invokeParam.Local)
	var wg sync.WaitGroup
	if exactly_once_intr.GuaranteeMth(baseQueryInput.GuaranteeMth) == exactly_once_intr.ALIGN_CHKPT {
		// InvokeRedisSetup(client, &RedisSetupInput{
		// 	FinalOutputTopicNames: params.SrcInvokeConfig.FinalTpNames,
		// }, invokeParam.GatewayUrl, invokeParam.Local)
		conn, err := checkpt.PrepareChkptClientGrpc()
		if err != nil {
			return nil, err
		}
		grpc_c := checkpt.NewChkptMngrClient(conn)
		_, err = grpc_c.Init(ctx, &checkpt.FinMsg{TopicNames: params.SrcInvokeConfig.FinalTpNames})
		if err != nil {
			return nil, err
		}
		chkMngrConfig := ChkptMngrInput{
			SrcTopicName:          params.SrcInvokeConfig.TopicName,
			FinalOutputTopicNames: params.SrcInvokeConfig.FinalTpNames,
			FinalNumOutPartitions: params.FinalOutPars,
			ChkptEveryMs:          baseQueryInput.CommitEveryMs,
			BufMaxSize:            baseQueryInput.BufMaxSize,
			SrcNumPart:            params.SrcInvokeConfig.NumOutPartition,
			GuaranteeMth:          baseQueryInput.GuaranteeMth,
			SerdeFormat:           baseQueryInput.SerdeFormat,
		}
		wg.Add(1)
		go InvokeChkMngr(&wg, client, &chkMngrConfig, invokeParam.GatewayUrl,
			invokeParam.Local)
		time.Sleep(time.Duration(5) * time.Millisecond)
	} else if exactly_once_intr.GuaranteeMth(baseQueryInput.GuaranteeMth) == exactly_once_intr.REMOTE_2PC {
		InvokeRTxnMngr(client, &RTxnMngrInput{SerdeFormat: baseQueryInput.SerdeFormat}, invokeParam.GatewayUrl, invokeParam.Local, 9)
	}

	fmt.Fprintf(os.Stderr, "src instance: %d\n", params.SrcInvokeConfig.NumSrcInstance)

	sourceOutput := InvokeSrc(&wg, client, params.SrcInvokeConfig, invokeSourceFunc, 1)

	time.Sleep(time.Duration(1) * time.Millisecond)

	outputMap := InvokeFunctions(&wg, client, params.CliNodes, params.InParamsMap, 1)
	fmt.Fprintf(os.Stderr, "Waiting for all client to return\n")
	wg.Wait()

	ParseSrcOutput(sourceOutput, invokeParam.StatDir)
	ParseFunctionOutputs(outputMap, invokeParam.StatDir)
	return params, nil
}

func InvokeDumpFunc(client *http.Client, dumpDir string,
	app_name string, serdeFormat commtypes.SerdeFormat, faas_gateway string,
	config map[string]uint8,
) {
	dumpInput := DumpStreams{
		DumpDir:     dumpDir,
		SerdeFormat: uint8(serdeFormat),
	}
	switch app_name {
	case "q1", "q2":
		dumpInput.StreamParams = []*StreamParam{
			{
				TopicName:  fmt.Sprintf("%s_out", app_name),
				KeySerde:   "String",
				ValueSerde: "Event",
			},
		}
	case "q3":
		dumpInput.StreamParams = []*StreamParam{
			{
				TopicName:  fmt.Sprintf("%s_out", app_name),
				KeySerde:   "Uint64",
				ValueSerde: "NameCityStateId",
			},
		}
	case "q4":
		dumpInput.StreamParams = []*StreamParam{
			{
				TopicName:  fmt.Sprintf("%s_out", app_name),
				KeySerde:   "Uint64",
				ValueSerde: "Float64",
			},
			{
				TopicName:  "q46_aucsByID",
				KeySerde:   "Uint64",
				ValueSerde: "Event",
			},
			{
				TopicName:  "q46_bidsByAucID",
				KeySerde:   "Uint64",
				ValueSerde: "Event",
			},
			{
				TopicName:  "q4_aucIDCat",
				KeySerde:   "AuctionIdCategory",
				ValueSerde: "AuctionBid",
			},
			{
				TopicName:  "q4_maxBids",
				KeySerde:   "Uint64",
				ValueSerde: "ChangeUint64",
			},
		}
	case "q6":
		dumpInput.StreamParams = []*StreamParam{
			{
				TopicName:  fmt.Sprintf("%s_out", app_name),
				KeySerde:   "Uint64",
				ValueSerde: "Float64",
			},
			{
				TopicName:  "q46_aucsByID",
				KeySerde:   "Uint64",
				ValueSerde: "Event",
			},
			{
				TopicName:  "q46_bidsByAucID",
				KeySerde:   "Uint64",
				ValueSerde: "Event",
			},
			{
				TopicName:  "q6_aucIDSeller",
				KeySerde:   "AuctionIdSeller",
				ValueSerde: "AuctionBid",
			},
			{
				TopicName:  "q6_maxBids",
				KeySerde:   "Uint64",
				ValueSerde: "ChangePriceTime",
			},
		}
	case "q5":
		dumpInput.StreamParams = []*StreamParam{
			{
				TopicName:  fmt.Sprintf("%s_out", app_name),
				KeySerde:   "StartEndTime",
				ValueSerde: "AuctionIdCntMax",
			},
			{
				TopicName:  "bids",
				KeySerde:   "Uint64",
				ValueSerde: "Event",
			},
			{
				TopicName:  "aucBids",
				KeySerde:   "StartEndTime",
				ValueSerde: "AuctionIdCount",
			},
		}
	case "q7":
		dumpInput.StreamParams = []*StreamParam{
			{
				TopicName:  fmt.Sprintf("%s_out", app_name),
				KeySerde:   "Uint64",
				ValueSerde: "BidAndMax",
			},
			{
				TopicName:  "bid_by_price",
				KeySerde:   "Uint64",
				ValueSerde: "Event",
			},
			{
				TopicName:  "bid_by_win",
				KeySerde:   "StartEndTime",
				ValueSerde: "Event",
			},
			{
				TopicName:  "max_bids",
				KeySerde:   "Uint64",
				ValueSerde: "StartEndTime",
			},
		}
	case "q8":
		dumpInput.StreamParams = []*StreamParam{
			{
				TopicName:  fmt.Sprintf("%s_out", app_name),
				KeySerde:   "Uint64",
				ValueSerde: "PersonTime",
			},
			{
				TopicName:  "q8_aucsBySellerID_out",
				KeySerde:   "Uint64",
				ValueSerde: "Event",
			},
			{
				TopicName:  "q8_personsByID_out",
				KeySerde:   "Uint64",
				ValueSerde: "Event",
			},
		}
	}
	for idx := range dumpInput.StreamParams {
		dumpInput.StreamParams[idx].NumPartitions = config[dumpInput.StreamParams[idx].TopicName]
	}
	url := BuildFunctionUrl(faas_gateway, "dump")
	fmt.Printf("func source url is %v\n", url)
	var response FnOutput
	if err := JsonPostRequest(client, url, "5", &dumpInput, &response); err != nil {
		log.Error().Msgf("dump request failed: %v", err)
	} else if !response.Success {
		log.Error().Msgf("dump request failed: %s", response.Message)
	}
	fmt.Fprintf(os.Stderr, "%sDump invoke done\n", app_name)
}
