package session

import (
	"os"
	"strings"
	"time"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/store"
)

type Event struct {
	At        time.Time `json:"at"`
	SessionID string    `json:"session_id"`
	State     string    `json:"state"`
}

func ReadEvents(path string) ([]Event, error) {
	f, e := store.OpenPrivate(path, os.O_RDONLY)
	if os.IsNotExist(e) {
		return []Event{}, nil
	}
	if e != nil {
		return nil, domain.Fail(4, "Cannot read private session events.", "Check state directory permissions.")
	}
	defer f.Close()
	var events []Event
	if e = store.DecodeStrict(f, 64*1024, &events); e != nil || len(events) > 256 {
		return nil, domain.Fail(2, "Private session events are invalid.", "Inspect private state storage.")
	}
	for _, event := range events {
		validState := event.State == "preflight" || event.State == "prepared" || event.State == "starting" || event.State == "listener_ready" || event.State == "stopping" || event.State == "idle"
		if !validState || event.At.IsZero() || len(event.SessionID) != 32 || strings.Trim(event.SessionID, "0123456789abcdef") != "" {
			return nil, domain.Fail(2, "Private session events are invalid.", "Inspect private state storage.")
		}
	}
	return events, nil
}
