package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

func copyAndClose(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

func handleTunneling(clientConn net.Conn, targetAddress string) {
	targetConn, err := net.DialTimeout("tcp", targetAddress, 10*time.Second)
	if err != nil {
		log.Println("Error connecting to target server:", err)
		clientConn.Close()
		return
	}

	clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	go copyAndClose(targetConn, clientConn)
	copyAndClose(clientConn, targetConn)
}

func handleClientRequest(clientConn net.Conn) {
	defer clientConn.Close()

	// Read and parse the client request
	buf := make([]byte, 4096)
	n, err := clientConn.Read(buf)
	if err != nil {
		log.Println("Error reading from client:", err)
		return
	}

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buf[:n])))
	if err != nil {
		log.Println("Error parsing client request:", err)
		return
	}

	if req.Method == http.MethodConnect {
		handleTunneling(clientConn, req.Host)
	} else {
		httpClient := &http.Client{}
		req.RequestURI = ""
		req.URL.Scheme = "http"
		req.URL.Host = req.Host

		resp, err := httpClient.Do(req)
		if err != nil {
			log.Println("Error forwarding request:", err)
			return
		}

		err = resp.Write(clientConn)
		if err != nil {
			log.Println("Error writing response to client:", err)
		}
	}
}

func main() {
	proxyAddress := ":8080"

	// Start listening for incoming connections
	proxyListener, err := net.Listen("tcp", proxyAddress)
	if err != nil {
		log.Fatal("Error starting proxy server:", err)
	}
	log.Println("Proxy server listening on", proxyAddress)

	// Accept incoming connections and handle them
	for {
		clientConn, err := proxyListener.Accept()
		if err != nil {
			log.Println("Error accepting client connection:", err)
			continue
		}

		go handleClientRequest(clientConn)
	}
}
