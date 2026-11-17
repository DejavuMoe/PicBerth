package core

import (
	"context"
	"testing"
	"time"

	"picberth/internal/meta"
)

func TestAuditSurvivesCancellationWhileWaitingForWriter(t *testing.T) {
	e := newEnv(t)
	conn, err := e.db.Writer().Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	waits := e.db.Writer().Stats().WaitCount
	go func() {
		e.svc.audit(ctx, e.admin, "test.disconnect", "", "")
		close(done)
	}()
	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for e.db.Writer().Stats().WaitCount == waits {
		select {
		case <-tick.C:
		case <-deadline:
			t.Fatal("audit never waited for the writer")
		}
	}
	cancel()
	conn.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("audit did not finish")
	}
	rows, err := e.db.ListAudit(context.Background(), 0, meta.AuditFilter{Action: "test.disconnect"}, 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("committed action lost its audit row: %+v, %v", rows, err)
	}
}
