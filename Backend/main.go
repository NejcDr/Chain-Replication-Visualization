package main

import(
	"fmt"
	"flag"
	"os"
	"bufio"
	"strings"
	"strconv"
)

var ORIGINAL_NUM_SERVERS int
var CURRENT_NUM_SERVERS int
var SERVERS []ServerConfig
var INIT_STORAGE []WriteObj
var RESET_MAIN_CHAN chan int
var RESET_CONTROL_CHAN chan int
var RESET_SERVERS_CHAN chan int

var NEXT_ID int
var EVENT_CHAN chan EventObj
var STEP int
var STEP_CHAN chan int
var TMP_STEP_ON bool
var TMP_STEP_CHAN chan int

func createServers() []ServerConfig {
	servers := make([]ServerConfig, CURRENT_NUM_SERVERS)
	saveChans := make([]chan WriteObj, CURRENT_NUM_SERVERS + 1)
	commitChans := make([]chan WriteObj, CURRENT_NUM_SERVERS + 1)
	getChans := make([]chan ReadObj, CURRENT_NUM_SERVERS)

	for i := 0; i < CURRENT_NUM_SERVERS; i++ {
		saveChans[i] = make(chan WriteObj, 10)
		commitChans[i] = make(chan WriteObj, 10)
		getChans[i] = make(chan ReadObj, 10)
	}

	saveChans[CURRENT_NUM_SERVERS] = nil
	commitChans[CURRENT_NUM_SERVERS] = nil
	tail_chan := getChans[CURRENT_NUM_SERVERS - 1]

	for i := 0; i < CURRENT_NUM_SERVERS; i++ {
		s := ServerConfig{
			Id: i, Storage: make(map[string]WriteObj),
			SAVE_CHANS: Duplex[WriteObj]{Input: saveChans[i], Output: saveChans[i+1]},
			COMMIT_CHANS: Duplex[WriteObj]{Input: commitChans[i+1], Output: commitChans[i]},
			GET_CHAN: getChans[i],
			TAIL_CHANS: Duplex[ReadObj]{Input: tail_chan, Output: make(chan ReadObj, 10)},
			PING_CHANS: Duplex[int]{Input: make(chan int), Output: make(chan int)},
			CONFIG_CHANS: Duplex[ServerConfig]{Input: make(chan ServerConfig), Output: make(chan ServerConfig)},
			STEP_SERVER_CHAN: make(chan int, 1), STEP_CONTROL_CHAN: make(chan int, 1),
			SHUTDOWN_CHAN: make(chan int),
			Head: i == 0, Tail: i == CURRENT_NUM_SERVERS - 1,
			New_Next: false, New_Prev: false,
			Sync: false, Sync_Continue: false}
		servers[i] = s
	}

	return servers
}

func initStorage(file_name string) []WriteObj {
	var init_storage []WriteObj

	if file_name != "" {
		file, err := os.Open(file_name)
		if err != nil {
			fmt.Println("Error:", err)
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			str := strings.Split(line, "|")

			key := strings.TrimSpace(str[0])
			value := strings.TrimSpace(str[1])
			version, _ := strconv.Atoi(strings.TrimSpace(str[2]))
			user := strings.TrimSpace(str[3])
			init_storage = append(init_storage, WriteObj{Key: key, Value: value, Version: version, User: user})
		}
		file.Close()
	}

	return init_storage
}

func startSimulation() {
	NEXT_ID = CURRENT_NUM_SERVERS
	EVENT_CHAN = make(chan EventObj, 100)
	STEP = 0
	STEP_CHAN = make(chan int, 1)
	TMP_STEP_ON = false
	TMP_STEP_CHAN = make(chan int, 1)

	go startServers()
	go startControl()
}

func startServers() {
	for i := 0; i < CURRENT_NUM_SERVERS; i++ {
		go Server(SERVERS[i])
	}

	<-RESET_SERVERS_CHAN
}

func startControl() {
	for i := 0; i < CURRENT_NUM_SERVERS; i++ { 
		go Ping(SERVERS[i]) 
	}

	<-RESET_CONTROL_CHAN
}

func resetSimulation() {
	RESET_CONTROL_CHAN <- 1
	RESET_SERVERS_CHAN <- 1
	CURRENT_NUM_SERVERS = ORIGINAL_NUM_SERVERS
	SERVERS = createServers()
	startSimulation()
}

func main() {
	var file_name string
	flag.IntVar(&ORIGINAL_NUM_SERVERS, "n", 5, "Defines initial number of servers in chain.")
	flag.StringVar(&file_name, "f", "", "File with initial storage. File must be .txt. Each line represents one value like: key | value | version | user")
	flag.Parse()

	if ORIGINAL_NUM_SERVERS < 1 && ORIGINAL_NUM_SERVERS > 7 {
		fmt.Println("Error: Number of servers must be at least 1 and at most 7.")
	}

	CURRENT_NUM_SERVERS = ORIGINAL_NUM_SERVERS
	SERVERS = createServers()
	INIT_STORAGE = initStorage(file_name)
	RESET_MAIN_CHAN = make(chan int)
	RESET_CONTROL_CHAN = make(chan int)
	RESET_SERVERS_CHAN = make(chan int)

	startSimulation()
	go StartRest()

	for {
		select {
		case <-RESET_MAIN_CHAN:
			resetSimulation()
		default:
			continue
		}
	}
}