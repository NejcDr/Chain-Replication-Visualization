package main

import(
	"fmt"
	"encoding/json"
    "log"
    "net/http"
	"time"

	"github.com/rs/cors"
)

func SendEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
        return
    }

	for {
		select {
		case step, ok := <-STEP_CHAN:
			if ok {
				time.Sleep(100 * time.Millisecond)
				events := []EventObj{}
				empty := false

				fmt.Printf("Step: %d\n", step)

				for !empty {
					select {
					case cmd, ok := <-EVENT_CHAN:
						if ok {
							events = append(events, cmd)
							fmt.Printf("Event: %v\n", cmd)
						} else {
							empty = true
						}
					default:
						empty = true
					}
				}

				eventsJSON := Events{Step: step, Events: events}
				data, err := json.Marshal(eventsJSON)
				if err != nil {
					http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
					return
				}

				msg := fmt.Sprintf("data: %s\n\n", data)
				fmt.Fprintf(w, msg)
				flusher.Flush()
			}
		}
	}
}

func Config(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ConfigurationObj{NServers: ORIGINAL_NUM_SERVERS, Init: INIT_STORAGE})
}

func Input(w http.ResponseWriter, r *http.Request) {
	write := WriteObj{}

	err := json.NewDecoder(r.Body).Decode(&write)
    if err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    SERVERS[0].SAVE_CHANS.Input <- write
    w.WriteHeader(http.StatusOK)
}

func Read(w http.ResponseWriter, r *http.Request) {
	read := ReadObj{}

	err := json.NewDecoder(r.Body).Decode(&read)
    if err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

	server, _ := findServer(read.Server)
	read.RETURN_CHAN = nil
	
	server.GET_CHAN <- read
	w.WriteHeader(http.StatusOK)
}

func Add(w http.ResponseWriter, r *http.Request) {
	ServerAdd()
	w.WriteHeader(http.StatusOK)
}

func Shutdown(w http.ResponseWriter, r *http.Request) {
	read := ReadObj{}

	err := json.NewDecoder(r.Body).Decode(&read)
    if err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

	server, _ := findServer(read.Server)
	server.SHUTDOWN_CHAN <- 1

	w.WriteHeader(http.StatusOK)
}

func Reset(w http.ResponseWriter, r *http.Request) {
	RESET_MAIN_CHAN <- 1
	w.WriteHeader(http.StatusOK)
}

func Step(w http.ResponseWriter, r *http.Request) {
	for i := 0; i < CURRENT_NUM_SERVERS; i++ {
		select {
		case SERVERS[i].STEP_SERVER_CHAN <- STEP:
		default:
			fmt.Printf("Failed to send clock signal to STEP_SERVER_CHAN of server %d\n", SERVERS[i].Id)
		}

		select {
		case SERVERS[i].STEP_CONTROL_CHAN <- STEP:
		default:
			fmt.Printf("Failed to send clock signal to STEP_CONTROL_CHAN of server %d\n", SERVERS[i].Id)
		}
	}
	
	select {
	case STEP_CHAN <- STEP:
	default:
		fmt.Println("Failed to send clock signal to global STEP_CHAN")
	}

	if TMP_STEP_ON {
		select {
		case TMP_STEP_CHAN <- STEP:
		default:
			fmt.Println("Failed to send clock signal to TMP_STEP_CHAN")
		}
	}

    STEP++
	w.WriteHeader(http.StatusOK)
}

func StartRest() {
	mux := http.NewServeMux()
	mux.HandleFunc("/events", SendEvents)
	mux.HandleFunc("/config", Config)
	mux.HandleFunc("/input", Input)
	mux.HandleFunc("/read", Read)
	mux.HandleFunc("/add", Add)
	mux.HandleFunc("/shutdown", Shutdown)
	mux.HandleFunc("/reset", Reset)
	mux.HandleFunc("/step", Step)

	handler := cors.Default().Handler(mux)

	log.Println("Starting server on :8080")
    err := http.ListenAndServe(":8080", handler)
    if err != nil {
        log.Fatalf("ListenAndServe: %v", err)
    }
}