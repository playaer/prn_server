package old

import (
	"os"
	"bufio"
	knq "github.com/knq/escpos"
	"strconv"
	"fmt"
	"os/exec"
	"io/ioutil"
	"strings"
	"github.com/tarm/serial"
	"time"
	"net"
	"io"
	"regexp"
	"sync"
	"net/http"
)

const (
	TypeUsb    string = "usb"
	TypeUsbCom string = "usb-com"
)

type Prn struct {
	Path string
	Type string
	Port int
	Err  error
}

func main1() {
	printServer()

	http.HandleFunc("/", hello)
	http.ListenAndServe(":9000", nil)
}

func hello(res http.ResponseWriter, req *http.Request) {
	res.Header().Set(
		"Content-Type",
		"text/html",
	)
	io.WriteString(
		res,
		`<doctype html>
<html>
	<head>
		<title>Hello World</title>
	</head>
	<body>
		Caffesta print server!
	</body>
</html>`,
	)
}

func printServer() {

	time.Sleep(10 * time.Second)

	contents, _ := ioutil.ReadDir("/dev")

	printers := []*Prn{}
	for _, f := range contents {
		if strings.Compare(f.Name(), "usb") == 0 {
			contents1, _ := ioutil.ReadDir("/dev/usb")
			for _, f1 := range contents1 {
				if strings.Contains(f1.Name(), "lp") {
					printers = append(printers, &Prn{Path: "/dev/usb/" + f1.Name(), Type: TypeUsb})
					fmt.Println(f1.Name())
				}
			}
		}
		if strings.Contains(f.Name(), "ttyUSB") {
			printers = append(printers, &Prn{Path: "/dev/" + f.Name(), Type: TypeUsbCom})
			fmt.Println(f.Name())
		}
	}

	for i, prnConfig := range printers {
		go server(prnConfig, i)
	}

	cmd := exec.Command("/bin/bash", "-c", "ifconfig")
	d, _ := cmd.Output()

	re := regexp.MustCompile(`(addr:(\d{1,3}\.?){4})`)
	addrs := re.FindAllString(string(d), -1)

	for _, prnConfig := range printers {
		go client(prnConfig, addrs)
	}
}

func server(prn *Prn, i int) {
	// listen on a port
	prn.Port = 9200 + i
	ln, err := net.Listen("tcp", "0.0.0.0:" + strconv.Itoa(prn.Port))
	if err != nil {
		prn.Err = err
		fmt.Println(err)
		return
	}

	m := new(sync.Mutex)
	for {
		// accept a connection
		c, err := ln.Accept()
		if err != nil {
			prn.Err = err
			fmt.Println(err)
			continue
		}
		// handle the connection
		go handleServerConnection(c, prn, m)
	}
}

func handleServerConnection(c net.Conn, prn *Prn, m *sync.Mutex) {

	m.Lock()
	defer m.Unlock()
	if prn.Type == TypeUsb {
		f, err := os.OpenFile(prn.Path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			prn.Err = err
		}
		io.Copy(f, c)
		time.Sleep(3 * time.Second)

		f.Close()
	} else {
		c1 := &serial.Config{Name: prn.Path, Baud: 19200, ReadTimeout: time.Second * 5}
		f, err := serial.OpenPort(c1)
		if err != nil {
			prn.Err = err
		}
		io.Copy(f, c)
		time.Sleep(3 * time.Second)

		f.Close()
	}

	c.Close()
}

func client(prn *Prn, addrs []string) {
	// connect to the server
	c, err := net.Dial("tcp", "127.0.0.1:" + strconv.Itoa(prn.Port))
	if err != nil {
		fmt.Println(err)
		return
	}

	w := bufio.NewWriter(c)
	p := knq.New(w)

	p.Init()
	p.SetFont("A")
	p.Write("https://caffesta.com")
	p.Formfeed()
	p.Formfeed()
	p.Write("Path: " + prn.Path)
	p.Formfeed()
	for i := range addrs {
		p.Write(addrs[i] + ":" + strconv.Itoa(prn.Port))
		p.Formfeed()
	}

	p.Formfeed()
	p.Cut()
	p.End()
	w.Flush()

	c.Close()
}
