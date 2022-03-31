package tools

import (
	"e2e/tools/common"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

type SimServer struct {
	*common.Node
	running bool
	srv     http.Server
}

func (server *SimServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Server", "APISIX Test Server")
	w.Write([]byte("response: " + server.IPPort()))
}

func (server *SimServer) Run() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		server.srv = http.Server{
			Addr:    server.IPPort(),
			Handler: server,
		}
		server.running = true
		fmt.Println("APISIX Test Server start: ", server.IPPort())
		wg.Done()
		server.srv.ListenAndServe()
		server.running = false
		fmt.Println("APISIX Test Server stop: ", server.IPPort())
	}()
	wg.Wait()
}
func (server *SimServer) Register(reg IRegCenter) {
	reg.Online(server.Node)
}

func (server *SimServer) LogOut(reg IRegCenter) {
	reg.Offline(server.Node)
}

func (server *SimServer) Stop() {
	server.srv.Close()
}

func (server *SimServer) Running() bool {
	return server.running
}

func NewSimServer(host, port, serviceName string) *SimServer {
	return &SimServer{
		Node: &common.Node{
			Host:        host,
			Port:        port,
			Weight:      1,
			ServiceName: serviceName,
			Args:        make(map[string]interface{}),
			Metadata:    make(map[string]interface{}),
		},
		running: false,
	}
}
func CreateSimServer(servers []*SimServer) error {
	for _, s := range servers {
		s.Run()
	}

	for _, s := range servers {
		if !s.Running() {
			return errors.New(fmt.Sprintf("APISIX Test Server start failed: %s", s.IPPort()))
		}
	}
	return nil
}
