package server

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
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
	s.uriMethodsMap[p] = append(s.uriMethodsMap[p], http.MethodGet)
	s.handlers[fmt.Sprintf("%s %s", http.MethodGet, p)] = h
}

func (s *Server) Post(p string, h http.Handler) {
	s.uriMethodsMap[p] = append(s.uriMethodsMap[p], http.MethodPost)
	s.handlers[fmt.Sprintf("%s %s", http.MethodPost, p)] = h
}

func (s *Server) Delete(p string, h http.Handler) {
	s.uriMethodsMap[p] = append(s.uriMethodsMap[p], http.MethodDelete)
	s.handlers[fmt.Sprintf("%s %s", http.MethodDelete, p)] = h
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

	// If the request is to large, return 413 TODO: Find better way to handle this
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

	var body io.Reader
	if len(reqSlice) > 1 {
		body = strings.NewReader(reqSlice[1])
	}
	request, err := http.NewRequest(method, uri, body)

	if err != nil {
		conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
		return
	}

	for i := 1; i < len(reqLineAndHeaders); i++ {
		header := strings.SplitN(reqLineAndHeaders[i], ":", 2)
		if len(header) != 2 {
			continue
		}
		key := strings.TrimSpace(header[0])
		values := strings.Split(strings.TrimSpace(header[1]), ", ")
		for _, value := range values {
			request.Header.Add(key, strings.TrimSpace(value))
		}
	}

	writer := NewWriter()
	s.handlers[fmt.Sprintf("%s %s", reqLine[0], reqLine[1])].ServeHTTP(writer, request)

	// Set Status if not set
	if writer.status == 0 {
		writer.status = http.StatusOK
	}

	// Set Content-Type if not set
	if writer.Header().Get("Content-Type") == "" {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}

	// Set Content-Length
	writer.Header().Set("Content-Length", strconv.Itoa(writer.body.Len()))

	// Set Date
	writer.Header().Set("Date", time.Now().UTC().Format(http.TimeFormat))

	// Set Connection, for now always close
	writer.Header().Set("Connection", "close")

	var headerStr string

	for key, values := range writer.Header() {
		for _, value := range values {
			headerStr = headerStr + fmt.Sprintf("%s: %s\r\n", key, value)
		}
	}

	conn.Write([]byte(fmt.Sprintf("%s %d %s\r\n%s\r\n%s", protocol, writer.status, http.StatusText(writer.status), headerStr, writer.body.String())))
}
