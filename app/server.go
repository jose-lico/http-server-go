package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"os"
	"slices"
	"strings"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	req := make([]byte, 1024)
	n, err := conn.Read(req)
	if err != nil {
		fmt.Println("Error reading from connection:", err)
		return
	}

	reqStr := string(req[:n])
	if reqStr == "" {
		return
	}

	reqSlice := strings.Split(reqStr, "\r\n\r\n")

	reqLineAndHeaders := strings.Split(reqSlice[0], "\r\n")

	reqLine := strings.Split(reqLineAndHeaders[0], " ")
	headers := make(map[string]string)
	body := reqSlice[1]

	for i := 1; i < len(reqLineAndHeaders); i++ {
		header := strings.SplitN(reqLineAndHeaders[i], ": ", 2)
		headers[header[0]] = header[1]
	}

	method := reqLine[0]
	path := reqLine[1]
	version := reqLine[2]

	if version != "HTTP/1.1" {
		conn.Write([]byte("HTTP/1.1 505 HTTP Version Not Supported\r\n\r\n"))
		return
	}

	if path == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		return
	} else if path == "/user-agent" {

		conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(headers["User-Agent"]), headers["User-Agent"])))
		return
	} else if strings.Split(path, "/")[1] == "files" {
		fileName := strings.Split(path, "/")[2]
		dir := os.Args[2]

		filePath := dir + fileName

		if method == "GET" {
			if _, err := os.Stat(filePath); err == nil {
				file, err := os.Open(filePath)
				if err != nil {
					conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
					return
				}
				defer file.Close()

				data, err := io.ReadAll(file)
				if err != nil {
					conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
					return
				}

				fileContent := string(data)

				conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(fileContent), fileContent)))
				return
			} else {
				conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
				return
			}
		} else if method == "POST" {
			err := os.WriteFile(filePath, []byte(body), 0644)

			conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
			if err != nil {
				conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
				return
			}
		} else {
			conn.Write([]byte("HTTP/1.1 403 Forbidden\r\n\r\n"))
			return
		}

	} else if strings.Split(path, "/")[1] == "echo" {
		message := strings.Split(path, "/")[2]
		encodings := strings.Split(headers["Accept-Encoding"], ", ")
		exists := slices.Contains(encodings, "gzip")
		if exists {
			compressed, err := compressString(message)
			if err != nil {
				conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
			}
			conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Encoding: gzip\r\nContent-Length: %d\r\n\r\n%s", len(compressed), compressed)))
		} else {
			conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(message), message)))
		}
		return
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return
	}
}

func compressString(s string) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)

	_, err := w.Write([]byte(s))
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
