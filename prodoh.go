package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/miekg/dns"
)

type flagStringList []string

func (i *flagStringList) String() string {
	return fmt.Sprint(*i)
}

func (i *flagStringList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	timeout   = flag.Duration("timeout", 10, "Timeout in seconds for the request to the DoH upstream server")
	address   = flag.String("address", ":5354", "Address to listen to (UDP)")
	upstreams flagStringList
)

func init() {
	flag.Var(&upstreams, "upstream", "List of upstream DoH servers")
}

// Question is dns query question
type Question struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

// Answer is dns query answer
type Answer struct {
	Name string `json:"name"`
	Type uint16 `json:"type"`
	TTL  int    `json:"TTL"`
	Data string `json:"data"`
}

// Response is dns query response
type Response struct {
	Status   int        `json:"Status"`
	TC       bool       `json:"TC"`
	RD       bool       `json:"RD"`
	RA       bool       `json:"RA"`
	AD       bool       `json:"AD"`
	CD       bool       `json:"CD"`
	Question []Question `json:"Question"`
	Answer   []Answer   `json:"Answer"`
}

func typeToString(dnsType uint16) (string, error) {
	switch dnsType {
	case dns.TypeA:
		return "A", nil
	case dns.TypeAAAA:
		return "AAAA", nil
	case dns.TypeCNAME:
		return "CNAME", nil
	case dns.TypeMX:
		return "MX", nil
	case dns.TypeTXT:
		return "TXT", nil
	case dns.TypeSPF:
		return "SPF", nil
	case dns.TypeNS:
		return "NS", nil
	case dns.TypeSOA:
		return "SOA", nil
	case dns.TypePTR:
		return "PTR", nil
	case dns.TypeANY:
		return "ANY", nil
	}
	return "", fmt.Errorf("Unsupported type %d", dnsType)
}

type QueryParams map[string]string

func httpGet(ctx context.Context, upstream string, params QueryParams) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", upstream, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/dns-json")
	q := req.URL.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func DoHQuery(upstream string, name string, dnsType uint16) ([]dns.RR, error) {
	t, err := typeToString(dnsType)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	if *timeout > 0 {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), *timeout*time.Second)
		defer cancel()
		ctx = ctxTimeout
	}

	params := QueryParams{
		"name": name,
		"type": strings.TrimSpace(string(t)),
	}
	body, err := httpGet(ctx, upstream, params)
	if err != nil {
		return nil, err
	}

	rr := &Response{}
	err = json.NewDecoder(bytes.NewBuffer(body)).Decode(rr)
	if err != nil {
		return nil, err
	}

	if rr.Status != 0 {
		return nil, fmt.Errorf("DOH failed response code %d", rr.Status)
	}

	var rrs = []dns.RR{}
	for _, answer := range rr.Answer {
		t, err := typeToString(answer.Type)
		if err != nil {
			return nil, err
		}
		s := fmt.Sprintf("%s %d IN %s %s", answer.Name, answer.TTL, t, answer.Data)
		rr, err := dns.NewRR(s)
		if err != nil {
			return nil, err
		}
		rrs = append(rrs, rr)
	}

	return rrs, nil
}

func parseQuery(m *dns.Msg, w dns.ResponseWriter, req *dns.Msg) {
	for _, q := range m.Question {
		for _, upstream := range upstreams {
			rsp, err := DoHQuery(upstream, q.Name, q.Qtype)
			if err != nil {
				log.Printf("%s DOH Query Failed: %v", upstream, err)
				continue
			}
			for _, rr := range rsp {
				m.Answer = append(m.Answer, rr)
			}
			return
		}
	}
	dns.HandleFailed(w, req)
}

func handleDnsRequest(w dns.ResponseWriter, req *dns.Msg) {
	if len(req.Question) == 0 {
		dns.HandleFailed(w, req)
		return
	}

	m := new(dns.Msg)
	m.SetReply(req)
	m.Compress = false

	switch req.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m, w, req)
	}

	w.WriteMsg(m)
}

func main() {
	flag.Parse()

	if len(upstreams) == 0 {
		log.Fatalf("-upstream is required")
	}

	server := &dns.Server{Addr: *address, Net: "udp"}
	dns.HandleFunc(".", handleDnsRequest)
	log.Printf("Listening at %v\n", *address)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

	// Wait for SIGINT or SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	server.Shutdown()
}
