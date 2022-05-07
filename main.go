package main

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
)

const help = `iss: A Gemini server
Usage:
iss [options]
Available options:
  -h
  --help		show this help
  -c
  --cert path	uses path as the certificate, default fullchain.pem
  -k
  --key path    uses path as the key, default privkey.pem
  -p
  --port port	uses port as receive port, default 1965
  -r
  --root path   uses path as the root of files, default static/
  
Examples:
  iss --help	show this help
  iss -c files/cert	listen on port 1965 using ./files/cert as certificate
`

func main() {
	port := "1965"
	cert := "fullchain.pem"
	key := "privkey.pem"
	root := "static/"
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-h", "--help":
			println(help)
			os.Exit(0)
		case "-p", "--port":
			i++
			port = os.Args[i]
		case "-c", "--cert":
			i++
			cert = os.Args[i]
		case "-k", "--key":
			i++
			key = os.Args[i]
		case "-r", "--root":
			i++
			root = os.Args[i]
		default:
			println("error: wrong argument", os.Args[i], "\n", help)
			os.Exit(-1)
		}
	}

	certficate, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		panic(err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{certficate},
	}
	tcp, err := net.Listen("tcp", ":"+port)
	if err != nil {
		panic(err)
	}
	defer tcp.Close()

	for {
		conn, err := tcp.Accept()
		if err != nil {
			println(err)
		}

		listener := tls.Server(conn, cfg)
		err = listener.Handshake()
		if err != nil {
			println(err.Error())
			conn.Close()
			continue
		}

		go handleConn(listener, root)
	}
}

func handleConn(c net.Conn, root string) {
	// read url
	req := make([]byte, 1024)
	n, err := c.Read(req)
	if err != nil {
		println("error reading request:", err.Error())
		return
	}

	reqURL, err := url.Parse(string(req[:n-2]))
	if err != nil {
		println("error parsing url:", err.Error())
		c.Write([]byte("59\r\n"))
		return
	}
	if reqURL.Path == "/" {
		reqURL.Path = "/index.gmi"
	}
	println("got request to", reqURL.Path)

	file := path.Join(root, reqURL.Path)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		println("error reading file:", err.Error())
		c.Write([]byte("51\r\n"))
		return
	}

	c.Write(append([]byte("20\r\n"), data...))
	c.Close()
}
