# HTTP Server in Go

## Project Description

Basic HTTP server in Go. While utilizing Go's standard `net` package for TCP connection handling, the core HTTP functionality is implemented from scratch.
This includes request parsing, method handling, response generation, and more.

## Features

- Uses Go's `net.Listen()` for TCP connection handling
- Custom implementation of:
  - HTTP request parsing
  - Handling of HTTP methods (GET, POST, etc.)
  - Request routing
    - Support for wildcards
    - Support for query params
  - Header and status code handling
  - HTTP response generation
  - Support for persistent connections (Keep-Alive)

## Resources

- [CodeCrafters HTTP server](https://app.codecrafters.io/courses/http-server/introduction)
- [HTTP/1.1 RFC](https://datatracker.ietf.org/doc/html/rfc2616)

[CodeCrafters referral link](https://app.codecrafters.io/r/graceful-shark-470603)
