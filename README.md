# prodoh
 
## Background
 ```prodoh``` is a DNS to DNS-over-HTTP Proxy. 
 
 ```prodoh``` listens to incoming DNS queries on a specified UDP port, converts the query to a DNS-over-HTTP query and forwards it to an upstream server, then converts back to a DNS answer.

 ```prodoh``` functions like ```cloudflared``` when used with the ```dns-proxy``` option.

## Install

Download the latest binary for your platform from the [releases page](https://github.com/poitch/prodoh/releases).

## Usage

```
prodoh -address 127.0.0.1:5353 \
    -upstream https://cloudflare-dns.com/dns-query \
    -upstream https://dns.google.com/resolve \
    -upstream https://dns9.quad9.net:5053/dns-query \
    -timeout 2
```


