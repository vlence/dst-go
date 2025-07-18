package simtest

import (
	"sync"
	"time"

	"github.com/vlence/gossert"
)

// A Timer represents an action that needs to be completed in the future.
// The interface is deliberately kept similar to that of *time.Timer.
type Timer interface {
        Reset(d time.Duration) bool

        Stop() bool
}

// SimTimer represents a simulater timer. Timers are used to represent
// a moment in the future. We can do something when a timer fires.
// SimTimer uses SimClock internally so the timer never fires unless
// the SimClock is Tick'ed and the timer's deadline is passed.
type SimTimer struct {
        ch       chan time.Time
        mu       *sync.Mutex
        stopped  bool
        deadline time.Time
        events   *timerEvents
}

// newSimTimer returns a new SimTimer.
func newSimTimer(deadline time.Time, events *timerEvents) *SimTimer {
        gossert.Ok(events != nil, "simtimer: timer events is nil")
        gossert.Ok(events.add != nil, "simtimer: add timer event is nil")
        gossert.Ok(events.remove != nil, "simtimer: remove timer event is nil")
        gossert.Ok(events.stop != nil, "simtimer: stop timer event is nil")
        gossert.Ok(events.tick != nil, "simtimer: tick timer event is nil")

        // Using buffer size 1 so that we aren't blocked if nobody is
        // waiting for a message from the channel yet. Size 1 is good
        // enough because we should be sending a message only once
        // using this channel. If we're sending a message more than
        // once it's a bug.
        ch := make(chan time.Time, 1)

        timer := new(SimTimer)
        timer.ch = ch
        timer.mu = new(sync.Mutex)
        timer.stopped = false
        timer.deadline = deadline
        timer.events = events

        return timer
}

// Reset updates the timer to fire after d time has passed
// since Reset is called. If the timer has already fired
// or was previously stopped then Reset does nothing.
func (timer *SimTimer) Reset(d time.Duration) bool {
        timer.mu.Lock()
        defer timer.mu.Unlock()

        if timer.stopped {
                return false
        }

        timer.deadline = timer.deadline.Add(d)

        return true
}

// Stop stops the timer. If the timer hasn't been fired when
// Stop is called then it'll never be fired. The timer's
// channel is not closed after the timer has been stopped.
func (timer *SimTimer) Stop() bool {
        timer.mu.Lock()
        defer timer.mu.Unlock()

        if timer.stopped {
                return false
        }

        timer.events.remove <- timer
        timer.stopped = true

        return true
}

// fire fires the timer if its deadline has been reached.
// The timer's deadline is reached if the current time now
// is after or equal to the deadline.
func (timer *SimTimer) fire(now time.Time) bool {
        timer.mu.Lock()
        defer timer.mu.Unlock()

        if timer.stopped {
                return false
        }

        passedDeadline := now.After(timer.deadline) || now.Equal(timer.deadline)

        if !passedDeadline {
                return false
        }

        timer.stopped = true
        timer.ch <- now

        return true
}
