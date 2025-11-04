package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			go handleConnection(conn)
		}
	}
}

func handleConnection(conn net.Conn) {
	r := bufio.NewReader(conn)
	defer conn.Close()
	headers := make(map[string]string)
	//read until \n\n
	for {
		line, err := r.ReadString('\n')
		if (len(line) == 2) || err != nil {
			break
		}
		tokens := strings.Split(line, ":")
		if len(tokens) < 2 {
			continue
		}
		headers[tokens[0]] = strings.TrimSpace(tokens[1])
	}
	upstream := "8081"
	if headers["x-client-version"] == "1" {
		upstream = "8082"
	} else if headers["x-client-version"] == "2" {
		upstream = "8083"
	}

	upstreamConn, err := net.Dial("tcp", "localhost:"+upstream)
	if err != nil {
		fmt.Printf("Error connecting to upstream: %s\n", err)
		return
	}
	defer upstreamConn.Close()
	for key, value := range headers {
		_, err := fmt.Fprintf(upstreamConn, "%s: %s\n", key, value)
		if err != nil {
			fmt.Printf("Error writing to upstream: %s\n", err)
			return
		}
	}
	_, err = fmt.Fprintf(upstreamConn, "\r\n")
	if err != nil {
		fmt.Printf("Error writing to upstream: %s\n", err)
		return
	}
	_, err = io.Copy(upstreamConn, conn)
	if err != nil {
		fmt.Printf("Error writing to upstream: %s\n", err)
		return
	}
	_, err = io.Copy(conn, upstreamConn)
	if err != nil {
		fmt.Printf("Error writing response from upstream: %s\n", err)
		return
	}

}
