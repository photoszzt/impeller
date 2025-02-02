package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sharedlog-stream/pkg/commtypes"
	"sync"

	"github.com/rs/zerolog/log"
)

func InvokeFunc(client *http.Client, response *FnOutput,
	wg *sync.WaitGroup, request interface{}, funcName string, gateway string,
) {
	defer wg.Done()
	url := BuildFunctionUrl(gateway, funcName)
	if err := JsonPostRequest(client, url, "", request, response); err != nil {
		log.Error().Err(err).Msgf("%s request failed", funcName)
	} else if !response.Success {
		log.Error().Msgf("%s request failed: %s", funcName, response.Message)
	}
}

func GetSerdeFormat(fmtStr string) commtypes.SerdeFormat {
	var serdeFormat commtypes.SerdeFormat
	if fmtStr == "json" {
		serdeFormat = commtypes.JSON
	} else if fmtStr == "msgp" {
		serdeFormat = commtypes.MSGP
	} else {
		log.Error().Msgf("serde format is not recognized; default back to JSON")
		serdeFormat = commtypes.JSON
	}
	return serdeFormat
}

func JsonPostRequest(client *http.Client, url string, nodeConstraint string, request interface{}, response interface{}) error {
	encoded, err := json.Marshal(request)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to encode JSON request")
		return fmt.Errorf("failed to encode JSON request: %v", err)
	}
	fmt.Printf("encoded json is %v, node constraint is %s\n", string(encoded), nodeConstraint)
	req, err := http.NewRequest("POST", url, bytes.NewReader(encoded))
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
	reader, err := DecompressFromReader(resp.Body)
	if err != nil {
		return err
	}
	if err := json.NewDecoder(reader).Decode(response); err != nil {
		log.Fatal().Err(err).Msg("failed to decode JSON response")
		return fmt.Errorf("failed to decode JSON response: %v", err)
	}
	return nil
}

func BuildFunctionUrl(gatewayAddr string, fnName string) string {
	return fmt.Sprintf("http://%s/function/%s", gatewayAddr, fnName)
}
