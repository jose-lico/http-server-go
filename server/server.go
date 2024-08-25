package server

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type Route struct {
	Pattern   []string
	Wildcards []string
	Handlers  map[string]http.Handler
}

type Server struct {
	routes []Route
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Get(p string, h http.Handler) {
	s.addRoute(http.MethodGet, p, h)
}

func (s *Server) Post(p string, h http.Handler) {
	s.addRoute(http.MethodPost, p, h)
}

func (s *Server) Delete(p string, h http.Handler) {
	s.addRoute(http.MethodDelete, p, h)
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

func (s *Server) addRoute(method, pattern string, handler http.Handler) {
	parts := strings.Split(pattern, "/")[1:]

	var wildcards []string

	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			wildcards = append(wildcards, part[1:len(part)-1])
			parts[i] = ""
		}
	}

	for _, route := range s.routes {
		if len(parts) != len(route.Pattern) {
			continue
		}

		match := true

		for i, part := range route.Pattern {
			if part != parts[i] {
				match = false
				break
			}
		}

		if match {
			route.Handlers[method] = handler
			return
		}
	}

	route := Route{
		Pattern:   parts,
		Wildcards: wildcards,
		Handlers:  make(map[string]http.Handler),
	}

	route.Handlers[method] = handler

	s.routes = append(s.routes, route)
}

func (s *Server) match(method, path string) (http.Handler, map[string]string, bool, bool) {
	parts := strings.Split(path, "/")[1:]

	for _, route := range s.routes {
		if len(parts) != len(route.Pattern) {
			continue
		}

		wildcards := make(map[string]string)
		match := true

		// Check pattern
		for i, part := range route.Pattern {
			if part == "" {
				// temp fix for index route edge case
				if len(route.Wildcards) == 0 {
					if part != parts[i] {
						match = false
					}

					break
				}

				wildcards[route.Wildcards[len(wildcards)]] = parts[i]
			} else if part != parts[i] {
				match = false
				break
			}
		}

		// Check method
		if match {
			handler, exists := route.Handlers[method]

			if exists {
				return handler, wildcards, true, true
			} else {
				return handler, wildcards, true, false
			}
		}
	}

	return nil, nil, false, false
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

	protocol := reqLine[2]

	// Only support protocol version HTTP/1.1, return 505 otherwise
	if protocol != "HTTP/1.1" {
		conn.Write([]byte("HTTP/1.1 505 HTTP Version Not Supported\r\n\r\n"))
		return
	}

	method := reqLine[0]
	uri, err := url.Parse(reqLine[1])

	if err != nil {
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	handler, wildcards, found, correctMethod := s.match(method, uri.Path)

	if !found {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return
	}

	if !correctMethod {
		conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
		return
	}

	var body io.Reader
	if len(reqSlice) > 1 {
		body = strings.NewReader(reqSlice[1])
	}
	request, err := http.NewRequest(method, uri.Path, body)

	for wildcard, value := range wildcards {
		request.SetPathValue(wildcard, value)
	}

	if err != nil {
		conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
		return
	}

	request.URL = uri

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
	handler.ServeHTTP(writer, request)

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
