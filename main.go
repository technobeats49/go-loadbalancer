package main

import (
	"fmt"
	"http"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	isAlive() bool
	Address() string
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	server          []Server
}

func newSimpleServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) isAlive() bool {
	return true
}

func (s *simpleServer) Serve(rw http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(rw, r)
}

func newLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		roundRobinCount: 0,
		port:            port,
		server:          servers,
	}
}

func handleErr(err error) {
	if err != nil {
		fmt.Errorf("error: %v\n", err)
		os.Exit(1)
	}
}

func (lb *LoadBalancer) getNextAvailable() Server {
	server := lb.server[lb.roundRobinCount%len(lb.server)]
	for !server.isAlive() {
		lb.roundRobinCount++
		server = lb.server[lb.roundRobinCount%len(lb.server)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) Server {
	targetServer := lb.getNextAvailable()
	fmt.Printf("forwarding requests to address %q\n", targetServer.Address())
	targetServer.Serve(rw, r)
}

func main() {
	serversList := []Server{
		newSimpleServer("https://www.github.com"),
		newSimpleServer("https://www.youtube.com"),
		newSimpleServer("https://www.twitter.com"),
	}

	lb := newLoadBalancer("9002", serversList)

	handleRedirect := func(rw http.ResponseWriter, r *http.Request) {
		lb.serveProxy(rw, r)
	}

	http.HandleFunc("/", handleRedirect)
	fmt.Printf("The server is serving requests at 'locahost:%s' \n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
