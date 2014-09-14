package main

import (
    "bufio"
    "fmt"
    "io"
    "log"
    "net"
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
    defer c.Close()
    dialer := &net.Dialer{Timeout: 100 * time.Millisecond}
    t, err := dialer.Dial("tcp", info.targetAddr)
    if err != nil {
        log.Fatal(fmt.Sprintf("can't open connection to %s", info.targetAddr))
    }
    defer t.Close()

    var buf = make([]byte, 4096)
    creader := bufio.NewReaderSize(c, 4096)
    // treader := bufio.NewReaderSize(t, 4096)
    for {
        if !transmit(creader, buf, t) {
            break
        }
        // if !transmit(treader, buf, c) {
        //     break
        // }
        time.Sleep(10 * time.Millisecond)
    }
}

func transmit(reader *bufio.Reader, buf []byte, t net.Conn) bool {
    n, err := reader.Read(buf)
    if n > 0 {
        t.Write(buf[0:n])
    }
    if err == io.EOF {
        return false
    }
    if err != nil {
        log.Fatal(err)
    }
    return true
}

func (proxy *ProxyInfo) connectionsAccepted() int {
    return 1
}
