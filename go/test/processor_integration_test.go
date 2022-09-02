/*
 * These integration tests expect a server running at TEST_BASE_URL
 * that is connected to both the default processor, and the httptarget
 * test target.
 */

package test

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

const (
	TEST_BASE_URL_VAR     = "TEST_BASE_URL"
	DEFAULT_TEST_BASE_URL = "http://localhost:10000"
)

func TestGets(t *testing.T) {
	type args struct {
		name         string
		path         string
		statusCode   int
		expectedBody string
		bodyChecker  func(*testing.T, []byte)
	}
	a := []args{
		{"Hello", "/hello", 200, "Hello, World!", nil},
		{"AddHeader", "/addHeader", 200, "Hello, World!", nil},
		{"NotFound", "/notfound", 404, "Not found", nil},
		{"GetToPost", "/getToPost", 200, "", isValidJson},
	}
	for _, arg := range a {
		t.Run(arg.name, func(t *testing.T) {
			resp, err := http.Get(makeUrl(t, arg.path))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if resp.StatusCode != arg.statusCode {
				t.Errorf("Incorrect status: want = %d, got = %d", arg.statusCode, resp.StatusCode)
			}
			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if arg.bodyChecker != nil {
				arg.bodyChecker(t, responseBody)
			} else if string(responseBody) != arg.expectedBody {
				t.Errorf("Incorrect response: want %q, got %q", arg.expectedBody, string(responseBody))
			}
		})
	}
}

func TestPosts(t *testing.T) {
	type args struct {
		name         string
		path         string
		contentType  string
		requestBody  string
		statusCode   int
		expectedBody string
	}
	encodeBody := "Encode this, please!"
	encodedBody := base64.StdEncoding.EncodeToString([]byte(encodeBody))

	a := []args{
		{"Echo", "/echo", "text/plain", "Hello, World!", 200, "Hello, World!"},
		{"CheckJSON", "/checkJson", "application/json", "{\"hello\": \"world\"}", 200, "{\"hello\": \"world\"}"},
		{"CheckNotJSON", "/checkJson", "application/json", "Seriosly, not even JSON", 400, "Invalid JSON"},
		{"CheckNotJSONButOK", "/checkJson", "text/plain", "Still not JSON", 200, "Still not JSON"},
		{"EchoEncode", "/echoencode", "text/plain", encodeBody, 200, encodedBody},
		{"EchoHashString", "/echohashstream", "text/plain", "Hash this!", 200, "Hash this!"},
	}
	for _, arg := range a {
		t.Run(arg.name, func(t *testing.T) {
			resp, err := http.Post(makeUrl(t, arg.path), arg.contentType, strings.NewReader(arg.requestBody))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if resp.StatusCode != arg.statusCode {
				t.Errorf("Incorrect status: want = %d, got = %d", arg.statusCode, resp.StatusCode)
			}
			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if string(responseBody) != arg.expectedBody {
				t.Errorf("Incorrect response: want %q, got %q", arg.expectedBody, string(responseBody))
			}
		})
	}
}

func TestAddHeader(t *testing.T) {
	resp, err := http.Get(makeUrl(t, "/addHeader"))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Incorrect status: want = 200, got = %d", resp.StatusCode)
	}
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	wantBody := "Hello, World!"
	if string(responseBody) != wantBody {
		t.Errorf("Incorrect response: want %q, got %q", wantBody, string(responseBody))
	}
	if resp.Header.Get("x-external-processor-status") == "" {
		t.Error("Expected value for x-external-processor-status")
	}
}

func makeUrl(t *testing.T, pathPart string) string {
	t.Helper()
	baseUrl := os.Getenv(TEST_BASE_URL_VAR)
	if baseUrl == "" {
		baseUrl = DEFAULT_TEST_BASE_URL
	}
	url, err := url.JoinPath(baseUrl, pathPart)
	if err != nil {
		t.Fatalf("Error joining URL: %v", err)
	}
	return url
}

func isValidJson(t *testing.T, body []byte) {
	t.Helper()
	if !json.Valid(body) {
		t.Fatalf("Test returned invalid JSON: %s", string(body))
	}
}
