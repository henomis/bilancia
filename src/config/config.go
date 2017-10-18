package config

import (
	"io/ioutil"
	"fmt"
	"encoding/json"
	"strings"
	"strconv"
	"sync"
	"sort"
)


const DefaultConfFile string = "./config/bilancia.conf"
//const defaultConfFile string = "/etc/bilancia.conf"

type Configuration struct {
	Name    string
	Port    int
	Servers []string
	Mode    int
}

type BalanceMap struct {
	Name   string
	Inport uint16
	//connCount map[uint16] uint32
	Servers   map[ServerName]uint32
	LastRound uint32
	Mode      int
	Mutex     sync.Mutex
}


const (
	ModeRoundRobin = iota
	ModeLeastConn
)

type ServerName struct {
	Host string
	Port uint16
}



func ReadConf(conf string) ([]BalanceMap, error) {

	bm := []BalanceMap{}

	/* read conf */
	file, err := ioutil.ReadFile(conf)
	if err != nil {
		fmt.Println("error:", err)
		return nil,err
	}

	var configuration []Configuration
	err = json.Unmarshal(file, &configuration)

	if err != nil {
		fmt.Println("error:", err)
		return nil,err
	}


	for c := range configuration {

		bm = append(bm, BalanceMap{})

		bm[c].Inport = uint16(configuration[c].Port)
		bm[c].Mode = configuration[c].Mode
		bm[c].Name = configuration[c].Name

		for s := range configuration[c].Servers {

			ss := strings.Split(configuration[c].Servers[s], ":")
			if len(ss) == 2 {
				bm[c].Init()
				p, _ := strconv.Atoi(ss[1])
				bm[c].AddServer(ServerName{ss[0], uint16(p)})
			}

		}

	}


	return bm, nil
}



func (m *BalanceMap) Init() {
	m.Servers = make(map[ServerName]uint32)

}

func (m *BalanceMap) AddConnection(port uint16) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	for k := range m.Servers {
		if k.Port == port {
			m.Servers[k]++
		}

	}
}

func (m *BalanceMap) DelConnection(port uint16) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	for k := range m.Servers {
		if k.Port == port {
			if m.Servers[k] > 0 {
				m.Servers[k]--
			}
		}

	}
}

func (m *BalanceMap) CountConnections(port uint16) uint32 {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	for k := range m.Servers {
		if k.Port == port {
			return m.Servers[k]
		}

	}
	return 0
}

func (m *BalanceMap) AddServer(server ServerName) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	m.Servers[server] = 0
}

func (m *BalanceMap) SelectConnection() ServerName {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	sel := ServerName{"", 0}

	switch m.Mode {

	case ModeLeastConn:
		var min uint32 = 0xffffffff
		for k, v := range m.Servers {
			if v < min {
				sel = k
				min = uint32(v)
			}
		}
	case ModeRoundRobin:

		port := 0
		var keys []int
		for k := range m.Servers {
			keys = append(keys, int(k.Port))
		}
		sort.Ints(keys)
		for i := range keys {
			if i == int(m.LastRound) {
				port = keys[(i+1)%len(keys)]
				m.LastRound = uint32((i + 1) % len(keys))
				break
			}
		}

		for k := range m.Servers {
			if k.Port == uint16(port) {
				sel = k
			}
		}

	}

	return sel
}
