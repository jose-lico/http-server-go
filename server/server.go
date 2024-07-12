package server

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type Server struct {
	handlers      map[string]http.Handler
	uriMethodsMap map[string][]string
}

func NewServer() *Server {
	return &Server{
		handlers:      make(map[string]http.Handler),
		uriMethodsMap: make(map[string][]string),
	}
}

func (s *Server) Get(p string, h http.Handler) {
	// TODO: check if method is already allowed on path
	s.uriMethodsMap[p] = append(s.uriMethodsMap[p], "GET")
	s.handlers[fmt.Sprintf("GET %s", p)] = h
}

func (s *Server) Post(p string, h http.Handler) {
	// TODO: check if method is already allowed on path
	s.uriMethodsMap[p] = append(s.uriMethodsMap[p], "POST")
	s.handlers[fmt.Sprintf("POST %s", p)] = h
}

func (s *Server) ListenAndServe(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// If the request is to large, return 403 TODO: Find better way to handle this
	req := make([]byte, 1024)
	n, err := conn.Read(req)
	if err != nil {
		fmt.Println("Error reading from connection:", err)
		conn.Write([]byte("HTTP/1.1 413 Payload Too Large\r\n\r\n"))
		return
	}

	reqStr := string(req[:n])
	if reqStr == "" {
		return
	}

	// Split Request Line and Headers from Body
	reqSlice := strings.Split(reqStr, "\r\n\r\n")
	// Split Request Line from Headers
	reqLineAndHeaders := strings.Split(reqSlice[0], "\r\n")
	// Split Method, URI and Version
	reqLine := strings.Split(reqLineAndHeaders[0], " ")

	method := reqLine[0]
	uri := reqLine[1]
	protocol := reqLine[2]

	// Only support protocol version HTTP/1.1, return 505 otherwise
	if protocol != "HTTP/1.1" {
		conn.Write([]byte("HTTP/1.1 505 HTTP Version Not Supported\r\n\r\n"))
		return
	}

	val, ok := s.uriMethodsMap[uri]

	if ok {
		// If the URI exists but the method is not allowed, return a 405
		if !contains(val, method) {
			conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
			return
		}

	} else {
		// If the URI does not exist, return a 404
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return
	}

	// Create header structure TODO: Should be map[string][]string
	// h := make(map[string][]string)
	// for i := 1; i < len(reqLineAndHeaders); i++ {
	// 	header := strings.SplitN(reqLineAndHeaders[i], ": ", 2)
	// 	h[header[0]] = header[1]
	// }

	body := []byte(reqSlice[1])
	request, err := http.NewRequest(method, uri, bytes.NewBuffer(body))

	if err != nil {
		conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
		return
	}

	writer := NewWriter()

	s.handlers[fmt.Sprintf("%s %s", reqLine[0], reqLine[1])].ServeHTTP(writer, request)
	conn.Write([]byte(fmt.Sprintf("%s 200 OK\r\n\r\n%s", protocol, writer.body.String())))
}
