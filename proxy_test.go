package main

import (
    "bufio"
    . "github.com/smartystreets/goconvey/convey"
    "io"
    "log"
    "net"
    "testing"
    "time"
)

type ActionResult struct {
    action string
    param1 int64
    param2 int
}
type resultChannel chan *ActionResult

type serverStub struct {
    listener   net.Listener
    connSource chan net.Conn
}

func acceptConnection(stub serverStub) {
    for {
        conn, err := stub.listener.Accept()
        if err != nil {
            log.Fatal("error on connection accept")
        }
        stub.connSource <- conn
        log.Println("connection was sent to channel")
    }
}

func runServerStub() serverStub {
    l, err := net.Listen("tcp", "localhost:0")
    if err != nil {
        log.Fatal("cannot bind on any port!", err)
    }
    stub := serverStub{l, make(chan net.Conn)}
    go acceptConnection(stub)
    return stub
}

func millisecondsSince(start time.Time) int64 {
    return int64(time.Since(start) / time.Millisecond)
}

func (stub *serverStub) acceptAndServeConnection(deadline time.Time) resultChannel {
    actions := make(resultChannel, 100)
    go stub.handleConnection(deadline, actions)
    return actions
}

func (stub *serverStub) handleConnection(deadline time.Time, actions resultChannel) {
    start := time.Now()
    select {
    case conn := <-stub.connSource:
        actions <- &ActionResult{"accepted", millisecondsSince(start), 0}
        log.Println("Got connection - serving")
        go serveConnection(conn, actions, deadline)
        return
    case <-time.After(deadline.Sub(time.Now())):
        return
    }
}

func timeMin(x, y time.Time) time.Time {
    if x.Before(y) {
        return x
    }
    return y
}

func serveConnection(conn net.Conn, results resultChannel, deadline time.Time) {
    start := time.Now()
    var buf = make([]byte, 4096)
    reader := bufio.NewReaderSize(conn, 1000)
    for {
        n, err := reader.Read(buf)
        if n > 0 {
            log.Println("read ", n)
            results <- &ActionResult{"read", millisecondsSince(start), n}
        }
        if err == io.EOF {
            results <- &ActionResult{"eof", millisecondsSince(start), 0}
            return
        }
        if err != nil {
            log.Fatal(err)
        }
        time.Sleep(10 * time.Millisecond)
    }
}
func getActivity(actions resultChannel, timeout time.Duration, expectedRecords int) []*ActionResult {
    records := make([]*ActionResult, 0, expectedRecords)
    for len(records) < expectedRecords {
        x := <-actions
        if x == nil {
            return records
        }
        records = append(records, x)
    }
    return records
}
func TestTestingTools(t *testing.T) {
    Convey("basic operation", t, func() {
        stub := runServerStub()
        results := stub.acceptAndServeConnection(time.Now().Add(2000 * time.Millisecond))

        dialer := &net.Dialer{Timeout: 100 * time.Millisecond}
        conn, _ := dialer.Dial("tcp", stub.listener.Addr().String())
        conn.Write([]byte("foo bar 1234567890\n"))

        time.AfterFunc(
            50*time.Millisecond,
            func() {
                conn.Write([]byte("foo bar 1234567890\n"))
                conn.Close()
            },
        )
        time.Sleep(2000 * time.Millisecond)
        close(results)
        activity := getActivity(results, 200*time.Millisecond, 10)
        So(activity[0].action, ShouldEqual, "accepted")
        So(activity[1].action, ShouldEqual, "read")
        So(activity[2].action, ShouldEqual, "read")
        So(activity[3].action, ShouldEqual, "eof")
        So(4, ShouldEqual, len(activity))
    })
}

func Test(t *testing.T) {
    Convey("basic operation", t, func() {
        stub := runServerStub()
        results := stub.acceptAndServeConnection(time.Now().Add(2000 * time.Millisecond))

        proxy := RunProxy(stub.listener.Addr().String(), "localhost:0")

        dialer := &net.Dialer{Timeout: 100 * time.Millisecond}
        conn, _ := dialer.Dial("tcp", proxy.addr)
        conn.Write([]byte("foo bar 1234567890\n"))

        time.AfterFunc(
            50*time.Millisecond,
            func() {
                conn.Write([]byte("foo bar 1234567890\n"))
                conn.Close()
            },
        )
        time.Sleep(2000 * time.Millisecond)
        close(results)
        activity := getActivity(results, 200*time.Millisecond, 10)
        So(activity[0].action, ShouldEqual, "accepted")
        So(activity[1].action, ShouldEqual, "read")
        So(activity[2].action, ShouldEqual, "read")
        So(activity[3].action, ShouldEqual, "eof")
        So(4, ShouldEqual, len(activity))

        So(1, ShouldEqual, proxy.connectionsAccepted())
    })
}
