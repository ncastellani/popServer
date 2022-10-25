package popserver

const (
	STATE_AUTHORIZATION = iota + 1 // connection just estabilished
	STATE_TRANSACTION              // authentication completed. default POP usage state
	STATE_UPDATE                   // connection to-be terminated (QUIT)
)
