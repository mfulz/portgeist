package protocol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// ReadRequest reads a single JSON request from the given reader.
// The JSON must be terminated by a newline.
func ReadRequest(r io.Reader) (*Request, error) {
	reader := bufio.NewReader(r)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}
	return &req, nil
}

// WriteRequest encodes and writes a Request to the given writer.
func WriteRequest(w io.Writer, req *Request) error {
	bytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encode error: %w", err)
	}
	bytes = append(bytes, '\n')
	_, err = w.Write(bytes)
	return err
}

// ReadResponse reads a single JSON response from the reader.
func ReadResponse(r io.Reader) (*Response, error) {
	reader := bufio.NewReader(r)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}
	return &resp, nil
}

// WriteResponse encodes and writes a Response to the writer.
func WriteResponse(w io.Writer, resp *Response) error {
	bytes, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("encode error: %w", err)
	}
	bytes = append(bytes, '\n')
	_, err = w.Write(bytes)
	return err
}
