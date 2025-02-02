package commtypes

type FailParam struct {
	FailAfterS uint32
	InstanceId uint8
	FailTimes  uint8
}

type FailSpec struct {
	FailSpec map[string]FailParam `json:"fail_spec"`
}
