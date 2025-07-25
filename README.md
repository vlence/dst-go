# Deterministic Simulation Testing in Go

This library provides some primitives for writing tests that deterministically simulate real world
faults for e.g.
- Corrupted data
- High latency

The usefulness of this library doesn't come from merely injecting these faults but being
able to reproduce them thereby being deterministic.

`simtest` is heavily inspired by TigerBeetle's VOPR.

If you're new to the idea of deterministic simulation testing I recommend you watch this video
of [ThePrimeagen talking with Joran Greef](https://www.youtube.com/watch?v=sC1B3d9C_sI).

## The Main Idea

Unit tests are great for testing the behaviour of your application. When you find a new bug you
can write another unit test to test for that bug's presence. If in the future you refactor or
add a new feature and the bug appears again the unit test will catch it. However unit tests cannot
find new bugs.

Fuzzing can be used to find new bugs. In case you don't know what fuzzing is it just means
generating random data and observing how your application behaves. Fuzzing is a great way to find
how well your application can handle data that it's not expecting. When new bugs are found by
fuzzing we can then create unit tests for them. 

To really put our application through the paces we need to be able to simulate the kind of faults
that can happen in the real world. In the real world tasks can take longer than expected, the
network is not reliable, disks fail, etc. One way to simulate these kind of faults is to use
`io.Reader` and `io.Writer` implementations that inject these faults. For example to simulate
high read latency we could wait for some time before starting the actual read operation.

```go
type HighLatencyReader struct {
    io.Reader
}

func NewHighLatencyReader(r io.Reader) *HighLatencyReader {
    return &HighLatencyReader{r}
}

func (r *HighLatencyReader) Read(p []byte) (int, error) {
    time.Sleep(1 * time.Second)
    return r.Reader.Read(p)
}
```

There's however the problem of time as well. Consider our previous example of introducing latency
into read operations. We are waiting for 1 second. If we are running this in our tests our test
will need to wait 1 second for the faulty read to complete. There's nothing wrong in this if we can
prove that our application works correctly or find new bugs but it is better if we can do all of
that and also not have to wait that long. 1 second is not a long time but there are a lot of
scenarios where you might have to wait much longer. For example you might have a response timeout
of 3s for HTTP requests. If you have 100s of endpoints in your application then that's a lot of
waiting!

It's fairly common to think of things like the network and disk as dependencies and to mock them
during tests. What is uncommon, as far as I know, is to think of time itself as a dependency.
If we can simulate time then we can make it go faster, fire timers and tickers sooner,
sleep faster, etc. A simple way to do this is to have an interface for a clock that can tell the
current time, create timers. sleep, etc. The clock can tick at a rate of our choosing and we
can make time go faster. Our earlier example can be rewritten like this:

```go
type HighLatencyReader struct {
    io.Reader
    clock Clock
}

func NewHighLatencyReader(r io.Reader, clock *Clock) *HighLatencyReader {
    return &HighLatencyReader{r, clock}
}

func (r *HighLatencyReader) Read(p []byte) (int, error) {
    clock.Sleep(1 * time.Second)
    return r.Reader.Read(p)
}
```

`clock` can tick at its own rate and thus make `Sleep` return much sooner than 1s.

### A Simulated Clock

`simtest` exports an interface `Clock` which can be used to tell the current time, sleep, create
timers, and create tickers. To simulate time it exports `SimClock` which is an implementation of
`Clock`. `SimClock` is entirely in control of the caller and does not progress unless `Tick` is
called.
