package main

import (
    "bufio"
    "flag"
    "fmt"
    "io"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"
)

type ProxyInfo struct {
    addr       string
    targetAddr string
}

func RunProxy(targetAddr string, listenOn string) *ProxyInfo {
    l, err := net.Listen("tcp", listenOn)
    if err != nil {
        log.Fatal(fmt.Sprintf("cannot bind on %s", listenOn), err)
    }
    proxyInfo := &ProxyInfo{l.Addr().String(), targetAddr}
    go proxyInfo.proxy(l)
    return proxyInfo
}

func (info *ProxyInfo) proxy(listener net.Listener) {
    connections := make(chan net.Conn)
    for i := 0; i < 5; i++ {
        go info.processConnections(connections)
    }
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Fatal("error on connection accept")
        }
        connections <- conn
    }
}

func (info *ProxyInfo) processConnections(connections chan net.Conn) {
    for {
        info.proxyConnection(<-connections)
    }

}

func (info *ProxyInfo) proxyConnection(c net.Conn) {
    dialer := &net.Dialer{Timeout: 1000 * time.Millisecond}
    t, err := dialer.Dial("tcp", info.targetAddr)
    if err != nil {
        log.Fatal(fmt.Sprintf("can't open connection to %s: %s", info.targetAddr, err))
        c.Close()
    }

    cClosed := make(chan bool)
    tClosed := make(chan bool)

    go proxyA2B(c, t, cClosed)
    go proxyA2B(t, c, tClosed)

    select {
    case <-cClosed:
        log.Println("source connection is closed. Closing target")
        t.Close()
    case <-tClosed:
        log.Println("target connection is closed. Closing source")
        c.Close()
    }
}

func proxyA2B(s, t net.Conn, sClosed chan bool) {
    buf := make([]byte, 4096)
    reader := bufio.NewReaderSize(s, 4096)
    for {
        n, err := reader.Read(buf)
        if n > 0 {
            t.Write(buf[0:n])
            log.Println(fmt.Sprintf("wrote %d bytes", n))
        }
        if err == io.EOF {
            sClosed <- true
            return
        }
        if err != nil {
            log.Println(fmt.Sprintf("read error: %s", err))
            return
        }
    }
}

func (proxy *ProxyInfo) connectionsAccepted() int {
    return 1
}

func main() {
    var local = flag.String("local", "localhost:9090", "local address host:port")
    var target = flag.String("target", "", "target address host:port")
    flag.Parse()
    if *target == "" {
        log.Fatal("target is required argument")
    }
    var p = RunProxy(*target, *local)
    fmt.Println(p)

    c := make(chan os.Signal, 1)
    signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
    sig := <-c
    if sig == syscall.SIGINT {
        fmt.Println()
    }
}
