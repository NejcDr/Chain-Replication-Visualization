package main

type WriteObj struct {
	Key string `json:"key"`
	Value string `json:"value"`
	Version int `json:"version"`
	Commited bool `json:"commited"`
	User string `json:"user"`
}

type ReadObj struct {
	Key string `json:"key"`
	Server int `json:"server"`
	RETURN_CHAN chan ReadObj `json:"-"`
}

type EventObj struct {
	Event string `json:"event"`
	Server int `json:"server"`
	Arguments []string `json:"arguments"`
}

type Events struct {
	Step int `json:"step"`
	Events []EventObj `json:"events"`
}

type ConfigurationObj struct {
	NServers int `json:"nservers"`
	Init []WriteObj `json:"init"`
}

type ServerConfig struct {
	Id int
	Storage map[string]WriteObj
	SAVE_CHANS Duplex[WriteObj]
	COMMIT_CHANS Duplex[WriteObj]
	GET_CHAN chan ReadObj
	TAIL_CHANS Duplex[ReadObj]
	PING_CHANS Duplex[int]
	CONFIG_CHANS Duplex[ServerConfig]
	STEP_SERVER_CHAN chan int
	STEP_CONTROL_CHAN chan int
	SHUTDOWN_CHAN chan int
	Head bool
	Tail bool
	New_Next bool
	New_Prev bool
	Resend []WriteObj
	Sync bool
	Sync_Continue bool
	Copy []WriteObj
}

type Duplex[T any] struct {
	Input chan T
	Output chan T
}