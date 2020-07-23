# prodoh
 
## Background
 ```prodoh``` is a DNS to DNS-over-HTTP Proxy. This programs listens to incoming DNS queries on a specified UDP ports, converts the query to a DNS-over-HTTP query and forwards it to an upstream servers, then converts back to a DNS answer.

 ```prodoh``` functions like ```cloudflared``` when used with the ```dns-proxy``` option.

## Usage

```
go run prodoh.go -address 127.0.0.1:5353 \
    -upstream https://cloudflare-dns.com/dns-query \
    -upstream https://dns.google.com/resolve \
    -upstream https://dns9.quad9.net:5053/dns-query \
    -timeout 2
```


