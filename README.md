# dynamic-proxyproto-go

A Go library implementation of the PROXY protocol, supporting both version 1 and version 2.

This project is another implementation of [github.com/pires/go-proxyproto](https://github.com/pires/go-proxyproto).

## Features
- PROXY protocol version 1/2
- TCP/UDP/UNIX
- IPv4/IPv6
- **Dynamic PROXY protocol listener (accepts both PROXY and non-PROXY connections)**
  - The main difference between this project and [go-proxyproto](https://github.com/pires/go-proxyproto).
