package fsm

/**
// FSMSnapshot is returned by an FSM in response to a Snapshot
// It must be safe to invoke FSMSnapshot methods with concurrent
// calls to Apply.
type FSMSnapshot interface {
	// Persist should dump all necessary state to the WriteCloser 'sink',
	// and call sink.Close() when finished or call sink.Cancel() on error.
	Persist(sink SnapshotSink) error

	// Release is invoked when we are finished with the snapshot.
	Release()
}
**/

import (
	"encoding/json"
	"log"

	"github.com/hashicorp/raft"
)

type fsmSnapshot struct {
	store map[string]string
}

func (snapshot *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	log.Println("Persist()") //
	err := func() error {
		// Encode data.
		b, err := json.Marshal(snapshot.store)
		if err != nil {
			return err
		}

		// Write data to sink.
		if _, err := sink.Write(b); err != nil {
			return err
		}

		// Close the sink.
		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
	}

	return err
}

func (snapshot *fsmSnapshot) Release() {
	log.Println("Release()") //
}
