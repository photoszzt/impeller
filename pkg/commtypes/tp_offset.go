package commtypes

type TpLogOff struct {
	Tp string
	// stores changelog offset for progress marking protocol;
	// stores the first marker seqnumber for align checkpoint
	LogOff uint64
}
