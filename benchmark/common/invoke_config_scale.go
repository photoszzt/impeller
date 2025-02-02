package common

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

func InvokeConfigScale(client *http.Client, csi *ConfigScaleInput,
	faas_gateway string, response *FnOutput, appName string, local bool,
) {
	url := BuildFunctionUrl(faas_gateway, appName)
	constraint := "1"
	if local {
		constraint = ""
	}
	if err := JsonPostRequest(client, url, constraint, csi, response); err != nil {
		log.Error().Msgf("%s request failed: %v", appName, err)
	} else if !response.Success {
		log.Error().Msgf("%s request failed: %s", appName, response.Message)
	}
}
