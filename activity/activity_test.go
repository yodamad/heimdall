package activity

import (
	"strconv"
	"sync"
	"testing"
)

func TestRingKeepsTheLastEntries(t *testing.T) {
	log := New(4)
	for i := 1; i <= 10; i++ {
		log.Info("", "line "+strconv.Itoa(i))
	}

	kept := log.Snapshot()
	if len(kept) != 4 {
		t.Fatalf("expected 4 entries kept, got %d", len(kept))
	}
	for i, e := range kept {
		wantSeq := uint64(7 + i)
		if e.Seq != wantSeq {
			t.Errorf("entry %d has sequence %d, expected %d", i, e.Seq, wantSeq)
		}
		if want := "line " + strconv.Itoa(7+i); e.Msg != want {
			t.Errorf("entry %d is %q, expected %q", i, e.Msg, want)
		}
	}
}

func TestSinceOnlyReturnsWhatIsNew(t *testing.T) {
	log := New(10)
	log.Info("", "first")

	entries, seq := log.Since(0)
	if len(entries) != 1 || entries[0].Msg != "first" {
		t.Fatalf("expected the first entry, got %v", entries)
	}

	if again, _ := log.Since(seq); len(again) != 0 {
		t.Errorf("expected nothing new, got %v", again)
	}

	log.Warn("repo", "second")
	log.Info("", "third")
	entries, _ = log.Since(seq)
	if len(entries) != 2 {
		t.Fatalf("expected 2 new entries, got %d", len(entries))
	}
	if entries[0].Msg != "second" || entries[0].Scope != "repo" || entries[0].Level != Warn {
		t.Errorf("unexpected first new entry : %+v", entries[0])
	}
	if entries[1].Msg != "third" {
		t.Errorf("unexpected second new entry : %+v", entries[1])
	}
}

func TestSinceLeavesAGapWhenLinesWereOverwritten(t *testing.T) {
	log := New(4)
	log.Info("", "read")
	_, seq := log.Since(0)

	for i := 0; i < 10; i++ {
		log.Info("", "line "+strconv.Itoa(i))
	}

	entries, _ := log.Since(seq)
	if len(entries) != 4 {
		t.Fatalf("expected the 4 entries kept, got %d", len(entries))
	}
	if entries[0].Seq <= seq+1 {
		t.Errorf("expected a gap after sequence %d, first entry is %d", seq, entries[0].Seq)
	}
}

func TestRecordPutsTheMessageOnOneLine(t *testing.T) {
	log := New(4)
	log.Info("", "  first\nsecond\r\nthird\tfourth\r  ")

	last, ok := log.Last()
	if !ok {
		t.Fatal("expected an entry")
	}
	if want := "first second third    fourth"; last.Msg != want {
		t.Errorf("message is %q, expected %q", last.Msg, want)
	}
}

func TestRecordIgnoresAnEmptyMessage(t *testing.T) {
	log := New(4)
	log.Info("", "   \n\t ")
	if _, ok := log.Last(); ok {
		t.Error("expected nothing to be recorded")
	}
	if entries := log.Snapshot(); len(entries) != 0 {
		t.Errorf("expected no entry, got %v", entries)
	}
}

func TestLastAndReset(t *testing.T) {
	log := New(4)
	if _, ok := log.Last(); ok {
		t.Error("expected no entry in a new log")
	}

	log.Info("", "one")
	log.Warn("repo", "two")
	last, ok := log.Last()
	if !ok || last.Msg != "two" || last.Level != Warn {
		t.Errorf("unexpected last entry : %+v", last)
	}

	log.Reset()
	if _, ok := log.Last(); ok {
		t.Error("expected no entry after a reset")
	}
	if entries := log.Snapshot(); len(entries) != 0 {
		t.Errorf("expected no entry after a reset, got %v", entries)
	}
}

func TestNewWithoutCapacityStillRecords(t *testing.T) {
	log := New(0)
	log.Info("", "one")
	log.Info("", "two")
	kept := log.Snapshot()
	if len(kept) != 1 || kept[0].Msg != "two" {
		t.Errorf("expected only the last entry, got %v", kept)
	}
}

// TestConcurrentRecordAndSince is the one which matters under -race : the workers record while the
// interface reads
func TestConcurrentRecordAndSince(t *testing.T) {
	log := New(64)

	const writers, each = 8, 1000
	var waitGroup sync.WaitGroup
	waitGroup.Add(writers)
	for w := 0; w < writers; w++ {
		go func(w int) {
			defer waitGroup.Done()
			for i := 0; i < each; i++ {
				log.Info("repo"+strconv.Itoa(w), "line "+strconv.Itoa(i))
			}
		}(w)
	}

	done := make(chan struct{})
	read := make(chan uint64, 1)
	go func() {
		var seq uint64
		var seen uint64
		for {
			entries, newSeq := log.Since(seq)
			for _, e := range entries {
				if e.Seq <= seq {
					t.Errorf("entry %d was returned again after sequence %d", e.Seq, seq)
				}
				seen++
			}
			seq = newSeq
			select {
			case <-done:
				read <- seen
				return
			default:
			}
		}
	}()

	waitGroup.Wait()
	close(done)
	<-read

	if _, ok := log.Last(); !ok {
		t.Error("expected entries to have been recorded")
	}
	if entries := log.Snapshot(); len(entries) != 64 {
		t.Errorf("expected the log to be full with 64 entries, got %d", len(entries))
	}
}
