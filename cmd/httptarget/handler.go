package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

const (
	kDefaultSize = 100
	kPattern     = "0123456789"
)

const helpMessage = `GET  /              : Print default message
GET  /help          : Print this message
POST /echo          : Echo back whatever you posted
GET  /json          : Return JSON content. Use optional "size" parameter for extra data.
GET  /json-trailers : Return JSON content with the trailer "x-test-target: Yes"
GET  /data?size=xxx : Return arbitrary characters xxx bytes long
`

type sampleJsonMessage struct {
	Testing       int    `json:"testing"`
	IsTesting     bool   `json:"isTesting"`
	HowTestyAreWe string `json:"howTestyAreWe"`
	ExtraData     []byte `json:"extraData,omitempty"`
}

func createHandler() http.Handler {
	router := httprouter.New()
	router.GET("/", handleIndex)
	router.GET("/help", handleHelp)
	router.GET("/hello", handleHello)
	router.POST("/echo", handleEcho)
	router.GET("/json", handleJson)
	router.GET("/json-trailers", handleJsonWithTrailers)
	router.GET("/data", handleData)
	return router
}

func handleIndex(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp.Header().Add("content-type", "text/plain")
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte("Use /help to find out what is possible\n"))
}

func handleHelp(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp.Header().Add("content-type", "text/plain")
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte(helpMessage))
}

func handleHello(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	resp.Header().Add("content-type", "text/plain")
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte("Hello, World!"))
}

func handleEcho(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	contentType := req.Header.Get("content-type")
	resp.Header().Add("content-type", contentType)
	io.Copy(resp, req.Body)
}

func handleJson(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	optionalSize := parseSize(req, 0)
	msg := sampleJsonMessage{
		Testing:       123,
		IsTesting:     true,
		HowTestyAreWe: "Very!",
	}
	if optionalSize > 0 {
		msg.ExtraData = makeData(optionalSize)
	}
	resp.Header().Add("content-type", "application/json")
	resp.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(resp)
	enc.Encode(&msg)
}

func handleJsonWithTrailers(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	msg := sampleJsonMessage{
		Testing:       123,
		IsTesting:     true,
		HowTestyAreWe: "Very!",
	}
	resp.Header().Add("content-type", "application/json")
	resp.Header().Add("Trailer", "x-test-target")
	resp.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(resp)
	enc.Encode(&msg)
	resp.Header().Add("x-test-target", "Yes")
}

func handleData(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	size := parseSize(req, kDefaultSize)
	resp.Header().Add("content-type", "application/octet-stream")
	resp.Header().Add("content-length", strconv.Itoa(size))
	resp.Write(makeData(size))
}

func parseSize(req *http.Request, defaultSize int) int {
	sizeStr := req.URL.Query().Get("size")
	if sizeStr == "" {
		return defaultSize
	} else {
		size, _ := strconv.Atoi(sizeStr)
		return size
	}
}

func makeData(size int) []byte {
	buf := bytes.Buffer{}
	for pos := 0; pos < size; pos += 10 {
		remaining := size - pos
		if remaining < 10 {
			buf.Write([]byte(kPattern[:remaining]))
		} else {
			buf.Write([]byte(kPattern))
		}
	}
	return buf.Bytes()
}
