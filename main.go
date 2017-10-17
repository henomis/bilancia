package main

import (
	"net"
	"fmt"
	"encoding/json"
	"sync"
	"strconv"
	"time"
	"sort"
	"os"
)


const (
	ModeRoundRobin = iota
	ModeLeastConn
)

type ServerName struct {
	host string
	port uint16
}

type balanceMap struct {
	inport uint16
	//connCount map[uint16] uint32
	servers map[ServerName] uint32
	lastRound uint32
	mode int
	mutex sync.Mutex
}

func (m *balanceMap) Init() {
	m.servers = make(map[ServerName] uint32)

}

func (m *balanceMap) AddConnection(port uint16 ) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for k := range m.servers {
		if k.port == port {
			m.servers[k]++
		}

	}
}

func (m *balanceMap) DelConnection(port uint16 ) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for k := range m.servers {
		if k.port == port {
			if m.servers[k] > 0 {
				m.servers[k]--
			}
		}

	}
}

func (m *balanceMap) CountConnections(port uint16 ) uint32 {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for k := range m.servers {
		if k.port == port {
			return m.servers[k]
		}

	}
	return 0
}


func (m *balanceMap) AddServer(server ServerName) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.servers[server] = 0
}




func (m *balanceMap) SelectConnection() ServerName {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	sel := ServerName{"",	0}

	switch m.mode {

	case ModeLeastConn:
		var min uint32 = 0xffffffff
		for k, v := range m.servers {
			if v < min {
				sel = k
				min = uint32(v)
			}
		}
	case ModeRoundRobin:

		port := 0
		var keys []int
		for k := range m.servers {
			keys = append(keys, int(k.port))
		}
		sort.Ints(keys)
		for i := range keys {
			if i == int(m.lastRound) {
				port = keys[(i+1) % len(keys)]
				m.lastRound = uint32((i+1) % len(keys))
				break
			}
		}

		for k := range m.servers {
			if k.port == uint16(port) {
				sel = k
			}
		}


	}



	return sel
}

func check(err error, message string) {
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", message)
}


func conn_job(external net.Conn, m *balanceMap) {

	defer external.Close()

	p := m.SelectConnection()
	fmt.Println("selected port: ",p)

	// select connection
	internal, err := net.Dial("tcp", p.host+":"+strconv.Itoa(int(p.port)))

	if err != nil {
		fmt.Println("Unable to connect to "+p.host+":"+strconv.Itoa(int(p.port)))
		return
	}

	m.AddConnection(p.port)


	check(err, "Connected to " +p.host+":"+strconv.Itoa(int(p.port)))
	defer m.DelConnection(p.port)
	defer internal.Close()



	// ext -> int
	go func(int net.Conn, ext net.Conn){
		buf :=make([]byte, 1024*10)
		for {
			for i := range buf {
				buf[i] = 0
			}
			n, err := ext.Read(buf)

			fmt.Printf("%d bytes from localhost:%d\n",n,m.inport)
			//fmt.Println(string(buf))

			if err != nil {
				fmt.Println("Disconnected on localhost:",m.inport)
				int.Close()
				break
			}

			int.Write(buf[:n])
		}
	}(internal,external)

	// ext <- int
	buf :=make([]byte, 1024*10)
	for {
		for i := range buf {
			buf[i] = 0
		}
		n, err := internal.Read(buf)
		fmt.Printf("%d bytes from %s:%d\n",n,p.host,p.port)

		//fmt.Println(string(buf))
		if err != nil {
			fmt.Printf("Disconnected on %s:%d",p.host,p.port)
			external.Close()
			break
		}

		external.Write(buf[:n])
	}

}

func manageTunnel(m *balanceMap) {

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(int(m.inport)))
	check(err, "[WS] Server is ready.")



	defer ln.Close()
	for {
		conn, err := ln.Accept()
		check(err, "[WS] Accepted connection.")
		go conn_job(conn,m)
	}



}


func main() {

	file, _ := os.Open("conf.json")
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(configuration)

	// settings...
	bm := [1000]balanceMap{}

	bm[0].Init()
	bm[0].AddServer(ServerName{"127.0.0.1",80})
	bm[0].AddServer(ServerName{"10.6.5.3",80})
	bm[0].inport = 9000
	bm[0].mode = ModeRoundRobin



	for i:= 0; i < len(bm); i++ {
		if(bm[i].inport != 0) {
			go manageTunnel(&bm[i])
		}
	}

	//wait
	for {
		time.Sleep(time.Second * 1)
	}


}
