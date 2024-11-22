package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/morzik45/go-queue/internal/server/handlers"
	"io"
	"net/http"
)

func main() {
	enqueue()
}

func enqueue() {
	ctx := context.Background()

	tm := map[string]interface{}{
		"test":  "test",
		"test2": 4,
	}

	var bJson []byte
	var err error

	b := handlers.EnqueueRequest{
		QueueType: "test",
		Priority:  3,
		Payload:   tm,
	}

	bJson, err = json.Marshal(b)
	if err != nil {
		panic(err)
	}

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, "POST", "http://localhost:8080/api/v1/enqueue", bytes.NewReader(bJson))
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	var resp *http.Response
	resp, err = funcName(client, req)
	if err != nil {
		panic(err)
	}

	defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)

	var result handlers.EnqueueResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		panic(err)
	}

	fmt.Println(result)
}

func funcName(client *http.Client, req *http.Request) (*http.Response, error) {
	return client.Do(req)
}
