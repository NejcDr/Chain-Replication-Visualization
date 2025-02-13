package main

func findServer(id int) (ServerConfig, int) {
	for index, server := range SERVERS {
		if server.Id == id {
			return server, index
		}
	}

	return ServerConfig{}, -1
}

func chanCopy(input chan WriteObj) []WriteObj {
	var storage []WriteObj

	for {
		select {
		case rec, _ := <-input:
			storage = append(storage, rec)
		default:
			return storage
		}
	}
}

func ServerAdd() {
	var tail ServerConfig
	var new ServerConfig

	tail = SERVERS[CURRENT_NUM_SERVERS-1]

	new = ServerConfig{
		Id: NEXT_ID, Storage: make(map[string]WriteObj),
		SAVE_CHANS: Duplex[WriteObj]{Input: make(chan WriteObj, 10), Output: nil},
		COMMIT_CHANS: Duplex[WriteObj]{Input: nil, Output: make(chan WriteObj, 10)},
		GET_CHAN: make(chan ReadObj, 10),
		TAIL_CHANS: Duplex[ReadObj]{Input: nil, Output: nil},
		PING_CHANS: Duplex[int]{Input: make(chan int), Output: make(chan int)}, 
		CONFIG_CHANS: Duplex[ServerConfig]{Input: make(chan ServerConfig), Output: make(chan ServerConfig)},
		STEP_SERVER_CHAN: make(chan int, 1), STEP_CONTROL_CHAN: make(chan int, 1),
		SHUTDOWN_CHAN: make(chan int), 
		Head: false, Tail: true, 
		New_Next: false, New_Prev: false,
		Sync: true, Sync_Continue: false}

	SERVERS = append(SERVERS, new)
	NEXT_ID++
	CURRENT_NUM_SERVERS++
	go Server(new)
    go Ping(new)

	_, indexOld := findServer(tail.Id)
	tail.Sync = true
	tail.SAVE_CHANS.Output = new.SAVE_CHANS.Input
	tail.COMMIT_CHANS.Input = new.COMMIT_CHANS.Output
	SERVERS[indexOld] = tail
	tail.CONFIG_CHANS.Input <- tail
	EVENT_CHAN <- EventObj{Event: "Modify Server (Sync New Tail)", Server: tail.Id}

	<-tail.CONFIG_CHANS.Output
	<-new.CONFIG_CHANS.Output

	_, indexNew := findServer(new.Id)
	new.Sync = false
	new.GET_CHAN = tail.GET_CHAN
	new.TAIL_CHANS.Input = tail.GET_CHAN
	SERVERS[indexNew] = new

	tail.Tail = false
	tail.Sync = false
	tail.TAIL_CHANS.Input = new.GET_CHAN
	tail.TAIL_CHANS.Output = make(chan ReadObj, 10)
	tail.GET_CHAN = make(chan ReadObj, 10)
	SERVERS[indexOld] = tail

	TMP_STEP_ON = true
	<-TMP_STEP_CHAN
	TMP_STEP_ON = false

	tail.CONFIG_CHANS.Input <- tail
	new.CONFIG_CHANS.Input <- new
	EVENT_CHAN <- EventObj{Event: "Modify Server (New Tail Synced)", Server: tail.Id}
	EVENT_CHAN <- EventObj{Event: "Modify Server (New Tail Synced)", Server: new.Id}
	return
}

func ServerDown(server ServerConfig) {
	var server_prev ServerConfig
	var server_next ServerConfig
	_, index := findServer(server.Id)

	if server.Head {
		server_next = SERVERS[index+1]
		server_next.Head = true
		SERVERS[index+1] = server_next

		server_next.CONFIG_CHANS.Input <- server_next
		EVENT_CHAN <- EventObj{Event: "Modify Server (New Head)", Server: server_next.Id}
	} else if server.Tail {
		server_prev = SERVERS[index-1]
		server_prev.Tail = true
		server_prev.SAVE_CHANS.Output = nil
		server_prev.COMMIT_CHANS.Input = nil
		server_prev.GET_CHAN = server.GET_CHAN
		SERVERS[index-1] = server_prev

		server_prev.CONFIG_CHANS.Input <- server_prev
		EVENT_CHAN <- EventObj{Event: "Modify Server (New Tail)", Server: server_prev.Id}
	} else {
		server_prev = SERVERS[index-1]
		server_next = SERVERS[index+1]
		server_prev.SAVE_CHANS.Output = server.SAVE_CHANS.Output
		server_next.COMMIT_CHANS.Output = server.COMMIT_CHANS.Output
		SERVERS[index-1] = server_prev
		SERVERS[index+1] = server_next
		server_prev.New_Next = true
		server_next.New_Prev = true
		server_prev.Resend = chanCopy(server.SAVE_CHANS.Input)
		server_next.Resend = chanCopy(server.COMMIT_CHANS.Input)

		server_prev.CONFIG_CHANS.Input <- server_prev
		server_next.CONFIG_CHANS.Input <- server_next
		EVENT_CHAN <- EventObj{Event: "Modify Server (New Next)", Server: server_prev.Id}
		EVENT_CHAN <- EventObj{Event: "Modify Server (New Previous)", Server: server_next.Id}
	}

	CURRENT_NUM_SERVERS--
	SERVERS = append(SERVERS[:index], SERVERS[index+1:]...)
	server.SHUTDOWN_CHAN <- 1
	return
}

func Ping(server ServerConfig) {
	for {
		if _, ok := <-server.STEP_CONTROL_CHAN; ok {
			cmd := EventObj{Event: "Ping", Server: server.Id}
			EVENT_CHAN <- cmd
			server.PING_CHANS.Input <- 0
		}

		timeout := 5
		recived := false

		for !recived && timeout > 0 {
			select {
			case <-server.PING_CHANS.Output:
				recived = true
			default:
				select {
				case <-server.PING_CHANS.Output:
					recived = true
				case _, ok := <-server.STEP_CONTROL_CHAN:
					if ok {
						timeout--
					}
				} 
			}
		}

		if !recived {
			ServerDown(server)
			return	
		}

		for i := 0; i < timeout - 1; i++ {
			select {
				case <-server.STEP_CONTROL_CHAN:
			}
		}
	}
}