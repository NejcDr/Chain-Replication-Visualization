package main

import(
	"fmt"
)

func (s *ServerConfig) InitStorage(init_storage []WriteObj) {
	var value WriteObj

	for _, rec := range init_storage {
		value.Key = rec.Key
		value.Value = rec.Value
		value.Version = rec.Version
		value.User = rec.User
		value.Commited = true

		s.Storage[value.Key] = value
	}
}

func (s *ServerConfig) Save(value WriteObj) (WriteObj, EventObj) {
	version := 0
	if v, ok := s.Storage[value.Key]; ok {
		version = v.Version
	}

	value.Commited = s.Tail
	value.Version = version + 1
	s.Storage[value.Key] = value

	event := map[bool]string{true: "Save (Tail)", false: "Save"}[s.Tail]
	str := fmt.Sprintf("Key: %s | Value: %s | Version: %d | Commited: %t | User: %s", value.Key, value.Value, value.Version, value.Commited, value.User)
	return WriteObj{Key: value.Key, Value: value.Value, Version: value.Version, Commited: value.Commited, User: value.User}, EventObj{Event: event, Server: s.Id, Arguments: []string{str}}
}

func (s *ServerConfig) Commit(value WriteObj) (WriteObj, EventObj) {
	v, _ := s.Storage[value.Key]

	if (v.Version <= value.Version) {
		v.Commited = true
		s.Storage[v.Key] = v
	}

	event := map[bool]string{true: "Commit", false: "Commit (Old)"}[v.Version <= value.Version]
	str := fmt.Sprintf("Key: %s | Value: %s | Version: %d | Commited: %t | User: %s", value.Key, value.Value, value.Version, value.Commited, value.User)
	return WriteObj{Key: value.Key, Value: value.Value, Version: value.Version, Commited: value.Commited, User: value.User}, EventObj{Event: event, Server: s.Id, Arguments: []string{str}}
}

func (s *ServerConfig) Get(read ReadObj) EventObj {
	var args []string

	if read.Key == "" {
		for k, v := range s.Storage {
			if !v.Commited {
				args = append(args, Ask(s.Id, k, s.TAIL_CHANS.Input, s.TAIL_CHANS.Output))
			} else {
				args = append(args, fmt.Sprintf("Key: %s | Value: %s | Version: %d | User: %s", k, v.Value, v.Version, v.User))
			}
		}
	} else {
		if v, ok := s.Storage[read.Key]; ok {
			if !v.Commited {
				args = append(args, Ask(s.Id, read.Key, s.TAIL_CHANS.Input, s.TAIL_CHANS.Output))
			} else {
				args = append(args, fmt.Sprintf("Key: %s | Value: %s | Version: %d | User: %s", read.Key, v.Value, v.Version, v.User))
			}
		}
	}

	return EventObj{Event: "Return", Server: s.Id, Arguments: args}
}

func Ask(id int, key string, tailGet chan ReadObj, tailReturn chan ReadObj) string {
	tailGet <- ReadObj{Key: key, Server: id, RETURN_CHAN: tailReturn}
	EVENT_CHAN <- EventObj{Event: "Ask", Server: id, Arguments: []string{fmt.Sprintf("Key: %s", key)}}
	returned, _ := <-tailReturn
	return returned.Key
}

func (s *ServerConfig) Answer(read ReadObj, return_server_id int) (ReadObj, EventObj) {
	if v, ok := s.Storage[read.Key]; ok {
		str := fmt.Sprintf("Key: %s | Value: %s | Version: %d | User: %s", v.Key, v.Value, v.Version, v.User)
		return ReadObj{Key: str, Server: return_server_id}, EventObj{Event: "Answer", Server: return_server_id, Arguments: []string{str}}
	}

	return ReadObj{Key: "", Server: return_server_id}, EventObj{Event: "Answer", Server: return_server_id}
}

func (s *ServerConfig) NewServerSave(value WriteObj) EventObj {
	if v, ok := s.Storage[value.Key]; !ok || v.Version <= value.Version {
		s.Storage[value.Key] = value
	}

	str := fmt.Sprintf("Key: %s | Value: %s | Version: %d | Commited: %t | User: %s", value.Key, value.Value, value.Version, value.Commited, value.User) 
	return EventObj{Event: "Sync (Save)", Server: s.Id, Arguments: []string{str}}
}

func (s *ServerConfig) CopyStorage() {
	s.Copy = make([]WriteObj, 0)

	for k, v := range s.Storage {
		if v.Commited {
			s.Copy = append(s.Copy, WriteObj{Key: k, Value: v.Value, Version: v.Version, Commited: v.Commited, User: v.User})
		}
	}
}