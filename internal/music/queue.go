package music

import (
	"fmt"
	"sync"
)

// DJEntry represents a track queued in the DJ auto-advance queue.
type DJEntry struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

var (
	djMu       sync.Mutex
	djQueue    []DJEntry
	DJFadeSecs = 4.0
	DjFading   bool
)

// DJAddToQueue searches the library for query and appends the first match to the DJ queue.
// Returns a description and the new queue length.
func DJAddToQueue(query string) (string, int, error) {
	tracks, err := Search(query)
	if err != nil || len(tracks) == 0 {
		return "", 0, fmt.Errorf("no track found for: %s", query)
	}
	t := tracks[0]
	entry := DJEntry{Title: t.Title, Artist: t.Artist}
	djMu.Lock()
	djQueue = append(djQueue, entry)
	pos := len(djQueue)
	djMu.Unlock()
	return fmt.Sprintf("%s – %s", entry.Title, entry.Artist), pos, nil
}

// DJClearQueue removes all tracks from the DJ queue.
func DJClearQueue() error {
	djMu.Lock()
	djQueue = djQueue[:0]
	djMu.Unlock()
	return nil
}

// DJMoveQueueItem reorders the DJ queue by moving the item at 1-based index from to 1-based index to.
func DJMoveQueueItem(from, to int) error {
	djMu.Lock()
	defer djMu.Unlock()
	n := len(djQueue)
	if from < 1 || from > n {
		return fmt.Errorf("from index %d out of range (1-%d)", from, n)
	}
	if to < 1 || to > n {
		return fmt.Errorf("to index %d out of range (1-%d)", to, n)
	}
	f := from - 1
	t := to - 1
	entry := djQueue[f]
	// Remove from original position
	djQueue = append(djQueue[:f], djQueue[f+1:]...)
	// Insert at new position (slice was shifted after removal)
	djQueue = append(djQueue[:t+1], djQueue[t:]...)
	djQueue[t] = entry
	return nil
}

// DJGetQueue returns a snapshot of the current DJ queue.
func DJGetQueue() []DJEntry {
	djMu.Lock()
	q := make([]DJEntry, len(djQueue))
	copy(q, djQueue)
	djMu.Unlock()
	return q
}

// DJQueueLen returns the number of items in the DJ queue.
func DJQueueLen() int {
	djMu.Lock()
	defer djMu.Unlock()
	return len(djQueue)
}

// DJDequeueNext removes and returns the next entry from the DJ queue.
// Returns false if the queue is empty or a fade transition is in progress.
func DJDequeueNext() (DJEntry, bool) {
	djMu.Lock()
	defer djMu.Unlock()
	if len(djQueue) == 0 || DjFading {
		return DJEntry{}, false
	}
	next := djQueue[0]
	djQueue = djQueue[1:]
	return next, true
}

// DJQueueCopy returns a copy of the queue slice under lock.
// Caller should use DJLock/DJUnlock for more complex operations.
func DJQueueCopy() []DJEntry {
	djMu.Lock()
	q := make([]DJEntry, len(djQueue))
	copy(q, djQueue)
	djMu.Unlock()
	return q
}
