package simtest

import (
	"math/rand/v2"
	"testing"
	"time"
)

func TestTimerHasExpectedDeadline(t *testing.T) {
        epoch := time.Now()
        clock := NewSimClock(epoch)
        defer clock.Stop()

        d := 1 * time.Second
        expectedDeadline := epoch.Add(d)

        tt, _ := clock.NewTimer(d)
        timer, _ := tt.(*SimTimer)

        if !expectedDeadline.Equal(timer.deadline) {
                t.Errorf("timer's deadline %s does not match expected deadline %s", timer.deadline, expectedDeadline)
        }
}

func TestTimerIsFiredAtDeadline(t *testing.T) {
        epoch := time.Now()
        clock := NewSimClock(epoch)
        defer clock.Stop()

        dur := 1 * time.Second
        _, ch := clock.NewTimer(dur)
        tt, ch := clock.NewTimer(dur)
        timer, _ := tt.(*SimTimer)

        tickSize := 100 * time.Millisecond
        minTicks := dur / tickSize
        iters := minTicks + 2
        for range iters {
                select {
                case now := <-ch:
                        if !now.Equal(timer.deadline) {
                                t.Errorf("timer fired at %s but should have been fired at %s", now, timer.deadline)
                        }
                        return
                default:
                        clock.Tick(tickSize)
                }
        }

        t.Errorf("timer wasn't fired")
}

func TestTimerIsFiredAfterDeadline(t *testing.T) {
        epoch := time.Now()
        clock := NewSimClock(epoch)
        defer clock.Stop()

        dur := 1 * time.Second
        tt, ch := clock.NewTimer(dur)
        timer, _ := tt.(*SimTimer)

        tickSize := 99 * time.Millisecond
        minTicks := dur / tickSize
        iters := minTicks + 3
        for range iters {
                select {
                case now := <-ch:
                        if !now.After(timer.deadline) {
                                t.Errorf("timer fired at %s but should have been fired after %s", now, timer.deadline)
                        }
                        return
                default:
                        clock.Tick(tickSize)
                }
        }

        t.Errorf("timer wasn't fired")
}

func TestTimerFiredOnlyOnce(t *testing.T) {
        epoch := time.Now()
        clock := NewSimClock(epoch)
        defer clock.Stop()

        dur := 1 * time.Second
        _, ch := clock.NewTimer(dur)

        fired := false
        tickSize := 100 * time.Millisecond
        for range 100 {
                select {
                case <-ch:
                        if fired {
                                t.Errorf("timer fired twice")
                        }

                        fired = true
                default:
                        clock.Tick(tickSize)
                }
        }

        if !fired {
                t.Errorf("timer not fired")
        }
}

func FuzzTimerFiredOnlyOnce(f *testing.F) {
        minResolution := int64(time.Microsecond)
        maxResolution := int64(time.Second)

        for range 10 {
                tickMul := rand.Int64N(100) + 1
                tickRes := rand.Int64N(maxResolution - minResolution) + minResolution
                tickSize := tickMul * tickRes

                durMul := rand.Int64N(1000)
                durRes := rand.Int64N(maxResolution - minResolution) + minResolution
                dur := durMul * durRes

                f.Add(tickSize, dur)
        }

        f.Fuzz(func(t *testing.T, a int64, b int64) {
                epoch := time.Now()
                clock := NewSimClock(epoch)
                defer clock.Stop()

                dur := time.Duration(b)
                tickSize := time.Duration(a)

                minTicks := (dur / tickSize) // min number of ticks before timer is fired
                maxIters := minTicks + 100

                _, ch := clock.NewTimer(dur)

                fired := false
                for range maxIters {
                        select {
                        case <-ch:
                                if fired {
                                        t.Errorf("timer fired twice")
                                }

                                fired = true
                        default:
                                clock.Tick(tickSize)
                        }
                }

                if !fired {
                        t.Errorf("timer not fired")
                }
        })
}

func FuzzClockWithMultipleTimers(f *testing.F) {
        minResolution := int64(time.Microsecond)
        maxResolution := int64(time.Second)

        for i := range uint(10) {
                tickMul := rand.Int64N(100) + 1
                tickRes := rand.Int64N(maxResolution - minResolution) + minResolution
                tickSize := tickMul * tickRes

                f.Add(i+1, int64(tickSize))
        }

        f.Fuzz(func(t *testing.T, numTimers uint, b int64) {
                epoch := time.Now()
                clock := NewSimClock(epoch)
                defer clock.Stop()

                tickSize := time.Duration(b)
                channels := make([]<-chan time.Time, numTimers)
                longestDur := int64(0)

                for i := range numTimers {
                        durMul := rand.Int64N(1000)
                        durRes := rand.Int64N(maxResolution - minResolution) + minResolution
                        dur := durMul * durRes

                        if longestDur < dur {
                                longestDur = dur
                        }

                        _, ch := clock.NewTimer(time.Duration(dur))

                        channels[i] = ch
                }

                minTicks := (longestDur / b) // min number of ticks before timer is fired
                maxIters := minTicks + 100

                fired := uint(0)
                for range maxIters {
                        for _, ch := range channels {
                                select {
                                case <-ch:
                                        fired++
                                default:
                                        ;
                                }
                        }

                        clock.Tick(tickSize)
                }

                if fired != numTimers {
                        t.Errorf("all timers were not fired")
                }
        })
}
