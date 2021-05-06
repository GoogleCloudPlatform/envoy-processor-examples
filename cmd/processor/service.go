package main

import (
	"encoding/json"
	"io"
	"regexp"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extproc_cfg "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_proc/v3alpha"
	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3alpha"
	envoy_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
)

var contentTypeJson = regexp.MustCompile("^(application|text)/json(;.*)?$")

type processorService struct{}

func (s *processorService) Process(stream extproc.ExternalProcessor_ProcessServer) error {
	msg, err := stream.Recv()
	if err == io.EOF {
		logger.Debug("Stream closed by proxy")
		return nil
	}
	if err != nil {
		logger.Errorf("Error receiving from stream: %s", err)
		return err
	}

	headers := msg.GetRequestHeaders()
	if headers == nil {
		logger.Warn("Expecting request headers message first")
		// Close stream since there's nothing else we can do
		return nil
	}

	path := getHeaderValue(headers.Headers, ":path")
	logger.Debugf("Received request headers for %s", path)

	switch path {
	case "/echo":
	case "/help":
	case "/hello":
	case "/json":
		// Pass through the basic ones so that the test target works as designed.
		// Just close the stream, which indicates "no more processing"
		return nil
	case "/addHeader":
		return processAddHeader(stream)
	case "/checkJson":
		return processCheckJson(stream, headers)
	case "/notfound":
		return processNotFound(stream)
	}

	// Do nothing for any unknown paths
	return nil
}

// Show how to return an error by returning a 404 in response to the "/notfound"
// path.
func processNotFound(stream extproc.ExternalProcessor_ProcessServer) error {
	// Construct an "immediate" response that will go back to the caller. It will
	// have:
	// * A status code of 404
	// * a content-type header of "text/plain"
	// * a body that says "Not found"
	// * an additional message that may be logged by envoy
	response := &extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_ImmediateResponse{
			ImmediateResponse: &extproc.ImmediateResponse{
				Status: &envoy_type.HttpStatus{
					Code: envoy_type.StatusCode_NotFound,
				},
				Headers: &extproc.HeaderMutation{
					SetHeaders: []*core.HeaderValueOption{
						{
							Header: &core.HeaderValue{
								Key:   "content-type",
								Value: "text/plain",
							},
						},
					},
				},
				Body:    "Not found",
				Details: "Requested path was not found",
			},
		},
	}
	// Send the message and return, which closes the stream
	return stream.Send(response)
}

// Show how to add a header to the response.
func processAddHeader(stream extproc.ExternalProcessor_ProcessServer) error {
	// Change the path to "/hello" because that's one of the paths that the target
	// server understands. (Sadly, go syntax and the way that it handles "oneof"
	// messages means a lot of boilerplate here!)
	err := stream.Send(&extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_RequestHeaders{
			RequestHeaders: &extproc.HeadersResponse{
				Response: &extproc.CommonResponse{
					HeaderMutation: &extproc.HeaderMutation{
						SetHeaders: []*core.HeaderValueOption{
							{
								Header: &core.HeaderValue{
									Key:   ":path",
									Value: "/hello",
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	msg, err := stream.Recv()
	if err != nil {
		return err
	}

	// We have not changed the processing mode, so the next message will be a
	// response headers message, and we should close the stream if it is not.
	responseHeaders := msg.GetResponseHeaders()
	if responseHeaders == nil {
		logger.Error("Expecting response headers as the next message")
		return nil
	}

	// Send back a response that adds a header
	response := &extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_ResponseHeaders{
			ResponseHeaders: &extproc.HeadersResponse{
				Response: &extproc.CommonResponse{
					HeaderMutation: &extproc.HeaderMutation{
						SetHeaders: []*core.HeaderValueOption{
							{
								Header: &core.HeaderValue{
									Key:   "x-external-processor-status",
									Value: "We were here",
								},
							},
						},
					},
				},
			},
		},
	}
	return stream.Send(response)
}

// This is a more sophisticated example that does a few things:
// 1) It sets a request header to rewrite the path
// 2) It checks the content-type header to see if it is JSON (or skips
//    this check if there is no content
// 3) If it is JSON, it changes the processing mode to get the request body
// 4) If the content is JSON, validate the request body as JSON
// 5) Return an error if the validation fails
// 6) Or, add a response header to reflect the request validation status
//
// Among other things, this example shows the ablility of an external processor
// to decide how much of the request and response it needs to process based
// on input.
func processCheckJson(stream extproc.ExternalProcessor_ProcessServer,
	requestHeaders *extproc.HttpHeaders) error {
	// Set the path to "/echo" to get the right functionality on the target.
	requestHeadersResponse := &extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_RequestHeaders{
			RequestHeaders: &extproc.HeadersResponse{
				Response: &extproc.CommonResponse{
					HeaderMutation: &extproc.HeaderMutation{
						SetHeaders: []*core.HeaderValueOption{
							{
								Header: &core.HeaderValue{
									Key:   ":path",
									Value: "/echo",
								},
							},
						},
					},
				},
			},
		},
	}

	// If the content-type header looks like JSON, then ask for the request body
	contentType := getHeaderValue(requestHeaders.Headers, "content-type")
	logger.Debugf("Checking content-type %s to see if it is JSON", contentType)
	contentIsJson := !requestHeaders.EndOfStream && contentTypeJson.MatchString(contentType)
	if contentIsJson {
		requestHeadersResponse.ModeOverride = &extproc_cfg.ProcessingMode{
			RequestBodyMode: extproc_cfg.ProcessingMode_BUFFERED,
		}
	}

	err := stream.Send(requestHeadersResponse)
	if err != nil {
		return err
	}

	var msg *extproc.ProcessingRequest
	var jsonStatus string

	if contentIsJson {
		msg, err = stream.Recv()
		if err != nil {
			return err
		}

		requestBody := msg.GetRequestBody()
		if requestBody == nil {
			logger.Error("Expected request body to be sent next")
			return nil
		}

		requestBodyResponse := &extproc.ProcessingResponse{}

		if json.Valid(requestBody.Body) {
			// Nothing to do, so return an empty response
			requestBodyResponse.Response = &extproc.ProcessingResponse_RequestBody{}

		} else {
			requestBodyResponse.Response = &extproc.ProcessingResponse_ImmediateResponse{
				ImmediateResponse: &extproc.ImmediateResponse{
					Status: &envoy_type.HttpStatus{
						Code: envoy_type.StatusCode_BadRequest,
					},
					Headers: &extproc.HeaderMutation{
						SetHeaders: []*core.HeaderValueOption{
							{
								Header: &core.HeaderValue{
									Key:   "content-type",
									Value: "text/plain",
								},
							},
						},
					},
					Body:    "Invalid JSON",
					Details: "Request body was not valid JSON",
				},
			}
			// Send the error response and end processing now
			return stream.Send(requestBodyResponse)
		}

		// Send the body response and wait for the next message
		err = stream.Send(requestBodyResponse)
		if err != nil {
			return err
		}
		jsonStatus = "Body is valid JSON"

	} else {
		jsonStatus = "Body is not JSON"
	}

	msg, err = stream.Recv()
	if err != nil {
		return err
	}

	responseHeaders := msg.GetResponseHeaders()
	if responseHeaders == nil {
		logger.Error("Expecting response headers as the next message")
		return nil
	}

	responseHeadersResponse := &extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_ResponseHeaders{
			ResponseHeaders: &extproc.HeadersResponse{
				Response: &extproc.CommonResponse{
					HeaderMutation: &extproc.HeaderMutation{
						SetHeaders: []*core.HeaderValueOption{
							{
								Header: &core.HeaderValue{
									Key:   "x-json-status",
									Value: jsonStatus,
								},
							},
						},
					},
				},
			},
		},
	}
	return stream.Send(responseHeadersResponse)
}

// getHeaderValue returns the value of the first HTTP header in the map that matches.
// We don't expect that we will need to look up many headers, so simply do a linear search. If
// we needed to query many more headers, we'd turn the header map into a "map[string]string",
// or a "map[string][]string" if we wanted to handle multi-value headers.
// Return the empty string if the header is not found.
func getHeaderValue(headers *core.HeaderMap, name string) string {
	for _, h := range headers.Headers {
		if h.Key == name {
			return h.Value
		}
	}
	return ""
}
