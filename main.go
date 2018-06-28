package main

import (
	"gopkg.in/gcfg.v1"
	"go.bug.st/serial.v1"
	"fmt"
	"net"
	"time"
	"io"
	"sync"
	"log"
	"strings"
)

type Connection struct {
	Port string
	Baud int
}

type Config struct {
	Connection Connection
}

var (
	cfg Config
)

func init() {
	err := gcfg.ReadFileInto(&cfg, "conf.gcfg")
	if err != nil {
		log.Fatal("Config error:", err)
	}
	
}

func main() {

	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	if len(ports) == 0 {
		fmt.Println("No serial ports found!")
	}

	// Print the list of detected ports
	found := false
	for _, port := range ports {
		fmt.Printf("Found port: %v\n", port)
		if strings.ToLower(port) == strings.ToLower(cfg.Connection.Port) {
			found = true
		}
	}
	if !found {
		fmt.Printf("NO PORTS FOUND!")
	}

	server()


}

func server() {
	ln, err := net.Listen("tcp", "0.0.0.0:9200")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Server started on ", ln.Addr().String())
	m := new(sync.Mutex)
	for {
		c, err := ln.Accept()
		c.SetDeadline(time.Now().Add(time.Second * 3))
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Print(".")
		go handleServerConnection(c, m)
	}
}

func handleServerConnection(c net.Conn, m *sync.Mutex) {
	m.Lock()
	defer func() {
		c.Close()
		m.Unlock()
	}()

	mode := &serial.Mode{
		BaudRate: cfg.Connection.Baud,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	f, err := serial.Open(cfg.Connection.Port, mode)

	//conn := &serial.Config{Name: cfg.Connection.Port, Baud: cfg.Connection.Baud, ReadTimeout: time.Second * 5}
	//f, err := serial.OpenPort(conn)
	if err != nil {
		fmt.Println("OpenPort", err)
		return
	}
	io.Copy(f, c)
	time.Sleep(3 * time.Second)
	f.Close()
}
