// Package activity keeps the last things Heimdall did, so that the interface can show what is
// going on while actions run on many repositories at once.
//
// It depends on the standard library only : every other package of Heimdall may record into it
// without an import cycle.
package activity

import (
	"strings"
	"sync"
	"time"
)

// Level says how much a line matters. Debug lines are kept, and left to the reader to hide : the
// store has no opinion on the verbosity asked for on the command line.
type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

// Entry is one thing which happened
type Entry struct {
	// Seq numbers the entries, from 1, and is never reused : a reader asks for what it has not
	// seen yet with it, and a jump tells it that lines were overwritten in between
	Seq   uint64
	At    time.Time
	Level Level
	// Scope is the repository the line is about, or empty when it is about the run itself
	Scope string
	Msg   string
}

// DefaultCapacity is the number of entries kept by the default log
const DefaultCapacity = 500

// Log keeps the last entries recorded, the oldest being dropped once it is full. Every method is
// safe to call from several goroutines : the workers record while the interface reads.
type Log struct {
	mu      sync.Mutex
	entries []Entry
	next    int    // where the next entry goes
	seq     uint64 // sequence given to the last entry recorded
	full    bool
}

// New returns a log keeping the last capacity entries. A capacity below 1 is read as 1.
func New(capacity int) *Log {
	if capacity < 1 {
		capacity = 1
	}
	return &Log{entries: make([]Entry, capacity)}
}

// flatten puts a message on a single line : an entry is rendered as one line, and the output of a
// command or of a git sideband is full of new lines and carriage returns
var flatten = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "\t", "    ")

// Record keeps a message. The caller is the one which strips the ANSI escapes, with
// utils.CleanForLog : the regexp lives in utils, which this package must not import.
func (l *Log) Record(level Level, scope, msg string) {
	msg = strings.TrimSpace(flatten.Replace(msg))
	if msg == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.seq++
	l.entries[l.next] = Entry{Seq: l.seq, At: time.Now(), Level: level, Scope: scope, Msg: msg}
	l.next = (l.next + 1) % len(l.entries)
	if l.next == 0 {
		l.full = true
	}
}

func (l *Log) Debug(scope, msg string) { l.Record(Debug, scope, msg) }
func (l *Log) Info(scope, msg string)  { l.Record(Info, scope, msg) }
func (l *Log) Warn(scope, msg string)  { l.Record(Warn, scope, msg) }
func (l *Log) Error(scope, msg string) { l.Record(Error, scope, msg) }

// Since returns the entries recorded after seq, oldest first, and the sequence to ask from the
// next time. When the first entry returned is not seq+1, the ones in between were overwritten
// because the reader was too slow : it is up to the caller to say so.
func (l *Log) Since(seq uint64) ([]Entry, uint64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	kept := l.kept()
	var entries []Entry
	for _, e := range kept {
		if e.Seq > seq {
			entries = append(entries, e)
		}
	}
	return entries, l.seq
}

// Snapshot returns every entry kept, oldest first
func (l *Log) Snapshot() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.kept()
}

// Last returns the entry recorded last, and false when nothing was recorded yet
func (l *Log) Last() (Entry, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.seq == 0 {
		return Entry{}, false
	}
	last := l.next - 1
	if last < 0 {
		last = len(l.entries) - 1
	}
	return l.entries[last], true
}

// Reset empties the log. It is there for the tests : nothing resets it while Heimdall runs.
func (l *Log) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = make([]Entry, len(l.entries))
	l.next, l.seq, l.full = 0, 0, false
}

// kept returns the entries in the order they were recorded. The lock has to be held.
func (l *Log) kept() []Entry {
	if !l.full {
		return append([]Entry(nil), l.entries[:l.next]...)
	}
	entries := make([]Entry, 0, len(l.entries))
	entries = append(entries, l.entries[l.next:]...)
	return append(entries, l.entries[:l.next]...)
}

// Default is the log Heimdall records into
var Default = New(DefaultCapacity)

// Record and the helpers below record into Default. They share their names with the levels and
// with the methods of Log, which Go allows : a method lives in the namespace of its type.
func Record(level Level, scope, msg string) { Default.Record(level, scope, msg) }
