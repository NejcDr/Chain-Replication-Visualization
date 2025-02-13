package main

import(
	"fmt"
	"time"
)

func save(s *ServerConfig, write WriteObj) {
	<-s.STEP_SERVER_CHAN
	write, event := s.Save(write)
		time.Sleep(10 * time.Millisecond)
	
		if s.Tail {
			s.COMMIT_CHANS.Output <- write
	
			if s.Sync {
				time.Sleep(10 * time.Millisecond)
				s.Copy = append(s.Copy, write)
			}
		} else {
			s.SAVE_CHANS.Output <- write
		}
	
		EVENT_CHAN <- event
		syncCheck(s)
}

func commit(s *ServerConfig, write WriteObj) {
	<-s.STEP_SERVER_CHAN
	if write.Key == "Sync (Confirm)" {
		s.Sync_Continue = true
		EVENT_CHAN <- EventObj{Event: "Sync (Continue)", Server: s.Id}
		return
	}

	write, event := s.Commit(write)
	if !s.Head {
		time.Sleep(10 * time.Millisecond)
		s.COMMIT_CHANS.Output <- write
	}

	EVENT_CHAN <- event
}

func get(s *ServerConfig, read ReadObj) {
	<-s.STEP_SERVER_CHAN
	done := make(chan int)
	defer close(done)

	EVENT_CHAN <- EventObj{Event: "Read", Server: s.Id}

	<-s.STEP_SERVER_CHAN
	go func() {
		var event EventObj
		var returned ReadObj

		if read.RETURN_CHAN != nil {
			returned, event = s.Answer(read, read.Server)
			read.RETURN_CHAN <- returned
		} else {
			event = s.Get(read)
			<-s.STEP_SERVER_CHAN
		}

		EVENT_CHAN <- event
		done <- 1
	}()

	finished := false
	for !finished {
		select {
		case <-done:
			finished = true
		case <-s.PING_CHANS.Input:
			pong(s)
		case <-s.STEP_SERVER_CHAN:
			continue
		}
	}

	syncCheck(s)
}

func pong(s *ServerConfig) {
	<-s.STEP_SERVER_CHAN
	s.PING_CHANS.Output <- 1
	EVENT_CHAN <- EventObj{Event: "Pong", Server: s.Id}
	syncCheck(s)
}

func shutdown(s *ServerConfig) {
	if _, ok := <-s.STEP_SERVER_CHAN; ok {
		EVENT_CHAN <- EventObj{Event: "Shutdown", Server: s.Id}
		for {
			select {
			case <-s.SHUTDOWN_CHAN:
				return
			default:
				select {
				case <-s.SHUTDOWN_CHAN:
					return
				case _, ok := <-s.PING_CHANS.Input:
					if ok { continue }
				case _, ok := <-s.STEP_SERVER_CHAN:
					if ok { continue }
				}		 
			}
		}
	}
}

func config(s *ServerConfig, newConfig ServerConfig) {
	<-s.STEP_SERVER_CHAN
	*s = newConfig
	EVENT_CHAN <- EventObj{Event: "Server Modified", Server: s.Id}
	if s.Sync {
		s.CopyStorage()
		s.Sync_Continue = true
	} else if s.New_Next {
		<-s.STEP_SERVER_CHAN
		for _, write := range s.Resend {
			str := fmt.Sprintf("Key: %s | Value: %s | Version: %d | Commited: %t | User: %s", write.Key, write.Value, write.Version, write.Commited, write.User)
			EVENT_CHAN <- EventObj{Event: "Save (Resend)", Server: s.Id, Arguments: []string{str}}
			s.SAVE_CHANS.Output <- write
		}
	} else if s.New_Prev {
		<-s.STEP_SERVER_CHAN
		for _, read := range s.Resend {
			str := fmt.Sprintf("Key: %s | Value: %s | Version: %d | Commited: %t | User: %s", read.Key, read.Value, read.Version, read.Commited, read.User)
			EVENT_CHAN <- EventObj{Event: "Commit (Resend)", Server: s.Id, Arguments: []string{str}}
			s.COMMIT_CHANS.Output <- read
		}
	}
}

func syncCheck(s *ServerConfig) {
	if s.Sync && s.Sync_Continue {
		s.Sync_Continue = false
		if len(s.Copy) >= 0 {
			syncSend(s)
		}
	}
}

func syncRecive(s *ServerConfig) {
	for {
		select {
		case <-s.PING_CHANS.Input:
			pong(s)
		default:
			select {
			case <-s.PING_CHANS.Input:
				pong(s)
			case v, _ := <-s.SAVE_CHANS.Input:
				<-s.STEP_SERVER_CHAN
				if v.Key == "" {
					EVENT_CHAN <- EventObj{Event: "Sync (Save)", Server: s.Id, Arguments: []string{"Sync (Done)"}}
					<-s.STEP_SERVER_CHAN
					s.CONFIG_CHANS.Output <- ServerConfig{}
					EVENT_CHAN <- EventObj{Event: "Sync (Done)", Server: s.Id}

					done := false
					for !done {
						select {
						case newConfig, _ := <-s.CONFIG_CHANS.Input:
							config(s , newConfig)
							done = true
						default:
							select {
							case newConfig, _ := <-s.CONFIG_CHANS.Input:
								config(s , newConfig)
								done = true
							case <-s.PING_CHANS.Input:
								pong(s)
							case <-s.STEP_SERVER_CHAN: 
							}
						}
					}
					return
				} else {
					cmd := s.NewServerSave(v)
					EVENT_CHAN <- cmd
				}

				<-s.STEP_SERVER_CHAN
				time.Sleep(10 * time.Millisecond)
				s.COMMIT_CHANS.Output <- WriteObj{Key: "Sync (Confirm)"}
				EVENT_CHAN <- EventObj{Event: "Sync (Confirm)", Server: s.Id}
			case <-s.STEP_SERVER_CHAN:
				continue
			}
		}
	}
}

func syncSend(s *ServerConfig) {
	time.Sleep(10 * time.Millisecond)

	if len(s.Copy) == 0 {
		s.SAVE_CHANS.Output <- WriteObj{}
		s.CONFIG_CHANS.Output <- ServerConfig{}
		EVENT_CHAN <- EventObj{Event: "Sync (Send)", Server: s.Id, Arguments: []string{"Sync (Done)"}}
		EVENT_CHAN <- EventObj{Event: "Sync (Done)", Server: s.Id}
		s.Copy = []WriteObj{}

		done := false
		for !done {
			select {
			case newServerConf, _ := <-s.CONFIG_CHANS.Input:
				config(s , newServerConf)
				done = true
			default:
				select {
				case newServerConf, _ := <-s.CONFIG_CHANS.Input:
					config(s , newServerConf)
					done = true
				case <-s.PING_CHANS.Input:
					pong(s)
				case <-s.STEP_SERVER_CHAN: 
				}
			}
		}

		return
	}

	s.SAVE_CHANS.Output <- s.Copy[0]
	str := fmt.Sprintf("Key: %s | Value: %s | Version: %d | Commited: true | User: %s", s.Copy[0].Key, s.Copy[0].Value, s.Copy[0].Version, s.Copy[0].User) 
	EVENT_CHAN <- EventObj{Event: "Sync (Send)", Server: s.Id, Arguments: []string{str}}

	s.Copy = s.Copy[1:]
}

func Server(serverConf ServerConfig) {
	s := &serverConf

	if s.Sync {
		syncRecive(s)
	} else {
		s.InitStorage(INIT_STORAGE)
	}
	
	for {
		select {
		case newConfig, _ := <-s.CONFIG_CHANS.Input:
			config(s , newConfig)
		default:
			select {
			case newConfig, _ := <-s.CONFIG_CHANS.Input:
				config(s , newConfig)
			case <-s.SHUTDOWN_CHAN:
				shutdown(s)
			case <-s.PING_CHANS.Input:
				pong(s)
			case write, _ := <-s.SAVE_CHANS.Input:
				save(s, write)
			case write, _ := <-s.COMMIT_CHANS.Input:
				commit(s, write)
			case read, _ := <-s.GET_CHAN:
				get(s, read)
			case <-s.STEP_SERVER_CHAN: 
				syncCheck(s)
			}
		}
	}
}