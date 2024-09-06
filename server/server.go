package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	MaxHeadersSize = 1024
	MaxRequestSize = 1024 * 2
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

	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		buffer := make([]byte, 1024)
		var req []byte
		var headersEnd int

		for {
			n, err := conn.Read(buffer)

			if err != nil {
				if err == io.EOF {
					return
				}

				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					fmt.Printf("Connection from %s timed out\n", conn.RemoteAddr().String())
					return
				}

				fmt.Println("Error reading from connection:", err)
				conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
				return
			}

			req = append(req, buffer[:n]...)

			if len(req) > MaxHeadersSize {
				conn.Write([]byte("HTTP/1.1 431 Request Header Fields Too Large\r\n\r\n"))
				if tcpConn, ok := conn.(*net.TCPConn); ok {
					tcpConn.CloseWrite()
				}
				return
			}

			headersEnd = bytes.Index(req, []byte("\r\n\r\n"))
			if headersEnd != -1 {
				break
			}
		}

		reqStr := string(req)

		// Split Request Line and Headers from Body
		reqSlice := strings.Split(reqStr, "\r\n\r\n")
		// Split Request Line from Headers
		reqLineAndHeaders := strings.Split(reqSlice[0], "\r\n")
		// Split Method, URI and Version
		reqLine := strings.Split(reqLineAndHeaders[0], " ")

		protocol := reqLine[2]

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

		headers := http.Header{}

		for i := 1; i < len(reqLineAndHeaders); i++ {
			header := strings.SplitN(reqLineAndHeaders[i], ":", 2)
			if len(header) != 2 {
				continue
			}
			key := strings.TrimSpace(header[0])
			values := strings.Split(strings.TrimSpace(header[1]), ", ")
			for _, value := range values {
				headers.Add(key, strings.TrimSpace(value))
			}
		}

		var contentLength int

		if _, exists := headers["Content-Length"]; exists {
			contentLengthStr := headers.Get("Content-Length")
			contentLength, err = strconv.Atoi(contentLengthStr)
			if err != nil {
				conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
				return
			}
		}

		bodyReadSoFar := len(reqSlice[1])
		remainingBodyLength := contentLength - bodyReadSoFar

		if contentLength > 0 {
			if len(req)+remainingBodyLength > MaxRequestSize {
				conn.Write([]byte("HTTP/1.1 413 Payload Too Large\r\n\r\n"))
				if tcpConn, ok := conn.(*net.TCPConn); ok {
					tcpConn.CloseWrite()
				}
				return
			}

			for remainingBodyLength > 0 {
				n, err := conn.Read(buffer)

				if err != nil {
					if err == io.EOF {
						return
					}

					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						fmt.Printf("Connection from %s timed out\n", conn.RemoteAddr().String())
						return
					}

					fmt.Println("Error reading from connection:", err)
					conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
					return
				}

				reqSlice[1] += string(buffer[:n])
				remainingBodyLength -= n

				if len(req)+n > MaxRequestSize {
					conn.Write([]byte("HTTP/1.1 413 Payload Too Large\r\n\r\n"))
					if tcpConn, ok := conn.(*net.TCPConn); ok {
						tcpConn.CloseWrite()
					}
					return
				}
			}
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
		request.Header = headers

		writer := NewWriter()

		if request.Header.Get("Connection") == "keep-alive" {
			writer.Header().Set("Connection", "keep-alive")
			writer.Header().Set("Keep-Alive", "timeout=5, max=100")
		} else {
			writer.Header().Set("Connection", "close")
		}

		handler.ServeHTTP(writer, request)

		if writer.status == 0 {
			writer.status = http.StatusOK
		}

		if writer.Header().Get("Content-Type") == "" {
			writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}

		writer.Header().Set("Content-Length", strconv.Itoa(writer.body.Len()))

		writer.Header().Set("Date", time.Now().UTC().Format(http.TimeFormat))

		var headerStr string

		for key, values := range writer.Header() {
			for _, value := range values {
				headerStr = headerStr + fmt.Sprintf("%s: %s\r\n", key, value)
			}
		}

		conn.Write([]byte(fmt.Sprintf("%s %d %s\r\n%s\r\n%s", protocol, writer.status, http.StatusText(writer.status), headerStr, writer.body.String())))

		if writer.Header().Get("Connection") != "keep-alive" {
			return
		}
	}
}
