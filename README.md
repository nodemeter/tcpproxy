tcpproxy
========

TCP Proxy with duplication (for dev and testing)

monitor POST requests to HOST
tcpdump -c 2000 -s 0 -A host HOST and tcp port http and 'tcp[32:4] = 0x504f5354'
