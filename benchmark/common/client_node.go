package common

import (
	"fmt"
	"net/http"
	"os"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/debug"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// ClientNode is used to construct the dag at the client; Each node is responsible to invoke the
// corresponding serverless function identified by the name
type ClientNode struct {
	config   *ClientNodeConfig
	children []*ClientNode
}

type ClientNodeConfig struct {
	GatewayUrl     string
	FuncName       string
	NodeConstraint string
	NumInstance    uint8
}

type InvokeParam struct {
	InputTopicName  string
	OutputTopicName string
	Duration        uint32 // in sec
	NumInPartition  uint16
	NumOutPartition uint16
	ParNum          uint16
	SerdeFormat     uint8
}

func NewClientNode(config *ClientNodeConfig) *ClientNode {
	return &ClientNode{
		config: config,
	}
}

func (n *ClientNode) Name() string {
	return n.config.FuncName
}

func (n *ClientNode) AddChild(node *ClientNode) {
	n.children = append(n.children, node)
}

func (n *ClientNode) Invoke(client *http.Client, response *FnOutput, wg *sync.WaitGroup,
	queryInput *QueryInput,
) {
	defer wg.Done()

	fmt.Fprintf(os.Stderr, "func name is %v\n", n.config.FuncName)
	url := BuildFunctionUrl(n.config.GatewayUrl, n.config.FuncName)
	fmt.Fprintf(os.Stderr, "func url is %s\n", url)

	// if err := JsonPostRequest(client, url, n.config.NodeConstraint, queryInput, response); err != nil {
	// 	log.Error().Msgf("%v request failed: %v", n.config.FuncName, err)
	// } else if !response.Success {
	// 	log.Error().Msgf("%v request failed with failed message: %v", n.config.FuncName, response.Message)
	// }
	i := 0
	startTime := time.Now()
	responses := make([]*FnOutput, 0, 2)
	failTimes := uint8(0)
	if queryInput.TestParams != nil {
		fmt.Fprintf(os.Stderr, "test params is %+v, func Name: %s\n", queryInput.TestParams, n.config.FuncName)
		failTimes = queryInput.TestParams[n.config.FuncName].FailTimes
	}
	for {
		var r FnOutput
		if i != 0 {
			// remove the test error on retry
			if queryInput.TestParams != nil {
				failTimes -= 1
				if failTimes == 0 {
					queryInput.TestParams = nil
				}
			}
		}
		err := JsonPostRequest(client, url, n.config.NodeConstraint, queryInput, &r)
		if err != nil {
			log.Error().Msgf("%v request failed: %v", n.config.FuncName, err)
			*response = r
			fmt.Fprintf(os.Stderr, "%s-%d call done\n", n.config.FuncName, queryInput.ParNum)
			return
		}
		if !r.Success {
			log.Error().Msgf("%v request failed with failed message: %v", n.config.FuncName, response.Message)
			*response = r
			fmt.Fprintf(os.Stderr, "%s-%d call done\n", n.config.FuncName, queryInput.ParNum)
			return
		} else {
			responses = append(responses, &r)
			debug.Fprintf(os.Stderr, "%s-%d msg: %s\n",
				n.config.FuncName, queryInput.ParNum, r.Message)
			if strings.Compare(r.Message, common_errors.ErrReturnDueToTest.Error()) == 0 {
				fmt.Fprintf(os.Stderr, "got test error and restart the failed instance now\n")
				if time.Since(startTime) >= time.Duration(queryInput.Duration)*time.Second {
					break
				}
				i += 1
				continue
			} else {
				break
			}
		}
	}
	if len(responses) != 0 {
		finalResponse := FnOutput{
			Success:   true,
			Counts:    make(map[string]uint64),
			Latencies: make(map[string][]int),
			EventTs:   make([]int64, 0),
		}
		for _, res := range responses {
			for s, n := range res.Counts {
				finalResponse.Counts[s] += n
			}
			for s, l := range res.Latencies {
				finalResponse.Latencies[s] = append(finalResponse.Latencies[s], l...)
			}
			finalResponse.EventTs = append(finalResponse.EventTs, res.EventTs...)
			finalResponse.Duration = res.Duration // the last one
		}
		*response = finalResponse
	}
	fmt.Fprintf(os.Stderr, "%s-%d call done\n", n.config.FuncName, queryInput.ParNum)
}
