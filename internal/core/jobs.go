package core

import (
	"time"
)

// StartJobs launches the in-process background jobs (docs/architecture.md §4): thumbnail worker with
// startup backfill, quick check 30 s after start and every 24 h, hourly purge of expired trash, hourly
// cleanup of expired sessions and hourly batched pruning of the audit log. Call once; Close stops them.
func (s *Service) StartJobs() {
	s.wg.Add(5)
	go s.runThumbWorker()
	go s.every(30*time.Second, 24*time.Hour, "quick check", func() {
		rep, err := s.Check(s.jobCtx, Principal{Kind: KindCLI, Username: "system"}, CheckQuick)
		if err != nil {
			s.log.Error("scheduled quick check failed", "err", err)
			return
		}
		if len(rep.Issues) > 0 {
			s.log.Warn("scheduled quick check found issues", "issues", len(rep.Issues), "ok", rep.OK)
		}
	})
	go s.every(45*time.Second, time.Hour, "trash purge", func() {
		if n, err := s.PurgeExpired(s.jobCtx); err != nil {
			s.log.Error("purging expired trash failed", "err", err)
		} else if n > 0 {
			s.log.Info("purged expired trash", "images", n)
		}
	})
	go s.every(time.Minute, time.Hour, "session cleanup", func() {
		if _, err := s.PurgeExpiredSessions(s.jobCtx); err != nil {
			s.log.Error("deleting expired sessions failed", "err", err)
		}
	})
	go s.every(2*time.Minute, time.Hour, "audit prune", func() {
		if n, err := s.PruneExpiredAudit(s.jobCtx); err != nil {
			s.log.Error("pruning the audit log failed", "err", err)
		} else if n > 0 {
			s.log.Info("pruned old audit entries", "entries", n)
		}
	})
}

// every runs fn after first, then each period, until the service closes.
func (s *Service) every(first, period time.Duration, name string, fn func()) {
	defer s.wg.Done()
	t := time.NewTimer(first)
	defer t.Stop()
	for {
		select {
		case <-s.jobCtx.Done():
			return
		case <-t.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						s.log.Error("background job panicked", "job", name, "panic", r)
					}
				}()
				fn()
			}()
			t.Reset(period)
		}
	}
}
