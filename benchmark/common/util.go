package common

import (
	"sharedlog-stream/pkg/commtypes"

	"github.com/rs/zerolog/log"
)

func StringToSerdeFormat(format string) commtypes.SerdeFormat {
	var serdeFormat commtypes.SerdeFormat
	if format == "json" {
		serdeFormat = commtypes.JSON
	} else if format == "msgp" {
		serdeFormat = commtypes.MSGP
	} else {
		log.Error().Msgf("serde format is not recognized; default back to JSON")
		serdeFormat = commtypes.JSON
	}
	return serdeFormat
}
