package entstore

import (
	"encoding/json"

	"github.com/wilhg/orch/pkg/store"
)

func structToEvent(id, runID, typ string, payload json.RawMessage) store.EventRecord {
	return store.EventRecord{
		EventID: id,
		RunID:   runID,
		Type:    typ,
		Payload: payload,
	}
}
