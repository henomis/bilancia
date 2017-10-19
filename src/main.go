package main

import (
	"./config"
	"fmt"

	"net"

	"strconv"

	"time"

	"flag"
)

func check(err error, message string) {
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", message)
}

func conn_job(external net.Conn, m *config.BalanceMap) {

	defer external.Close()

	p := m.SelectConnection()
	fmt.Println("selected port: ", p)

	// select connection
	internal, err := net.Dial("tcp", p.Host+":"+strconv.Itoa(int(p.Port)))

	if err != nil {
		fmt.Println("Unable to connect to " + p.Host + ":" + strconv.Itoa(int(p.Port)))
		return
	}

	m.AddConnection(p)

	check(err, "Connected to "+p.Host+":"+strconv.Itoa(int(p.Port)))
	defer m.DelConnection(p)
	defer internal.Close()

	// ext -> int
	go func(int net.Conn, ext net.Conn) {
		buf := make([]byte, 1024*32)
		for {
			for i := range buf {
				buf[i] = 0
			}
			n, err := ext.Read(buf)

			fmt.Printf("%d bytes from localhost:%d\n", n, m.Inport)
			//fmt.Println(string(buf))

			if err != nil {
				fmt.Println("Disconnected on localhost:", m.Inport)
				int.Close()
				break
			}

			int.Write(buf[:n])
		}
	}(internal, external)

	// ext <- int
	buf := make([]byte, 1024*32)
	for {
		for i := range buf {
			buf[i] = 0
		}
		n, err := internal.Read(buf)
		fmt.Printf("%d bytes from %s:%d\n", n, p.Host, p.Port)

		//fmt.Println(string(buf))
		if err != nil {
			fmt.Printf("Disconnected on %s:%d", p.Host, p.Port)
			external.Close()
			break
		}

		external.Write(buf[:n])
	}

}

func manageTunnel(m *config.BalanceMap) {

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(int(m.Inport)))
	if err != nil {
		panic(err)
	}
	fmt.Printf("[%s] Server is ready!\n", m.Name)

	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		fmt.Printf("[%s] Accepted connection!\n", m.Name)

		go conn_job(conn, m)
	}

}

func main() {

	c := flag.String("conf",config.DefaultConfFile,"conf file")
	flag.Parse()

	bm, err := config.ReadConf(*c)
	if(err != nil) {
		panic(err)
	}


	// lanch goroutine
	for i := 0; i < len(bm); i++ {
		if bm[i].Inport != 0 {
			go manageTunnel(&bm[i])
		}
	}

	//wait
	for {
		time.Sleep(time.Second * 1)
	}

}
