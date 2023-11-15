"cmd/processor" contains sample external processor, written in Go,
that exercise most of the functionality of ext_proc, including:

* Reading and validating header content
* Modifying request and response headers
* Reading the request body using several different streaming modes
* Modifying the request body using the different streaming modes

In addition, "cmd/httptarget" contains an HTTP server that can be used as an
"upstream" server, along with a sample configuration that runs Envoy, the sample
target, and the processor.

## Comprehensive Example

Follow these instructions to run all the parts together to exercise all the
functionality.

### Build and Start the HTTP target

Build the HTTP target and run it in the background on port 10001:

    cd go
    go build ./cmd/httptarget
    ./httptarget -p 10001 &

You can verify this using "curl":

    curl 0:10001/hello
    Hello, World!

### Build and Start the Go processor

Build the external processing server and run it in the background on port 10002:

    cd go
    go build ./cmd/processor
    ./processor -p 10002 &

Alternately, the processor may be run listening on a unix socket

    ./processor -s /tmp/processor.sock &

### Get Envoy

You'll need an Envoy proxy binary in order to test the processor. The
[Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/start/install)
has many options listed. You can also build Envoy from source.

### Start Envoy

Run Envoy using the supplied configuration, which will:

* Listen on port 10000
* Proxy all requests to the HTTP target at port 10001
* Pass all requests through the processor for additional processing

Assuming that "envoy" points to an Envoy executable in your environment,
then run the following from this directory:

    envoy -c envoy.yaml

### Test the flow

Ensure that a standard HTTP request gets through Envoy, the processor, to the HTTP target,
and back again:

    curl 0:10000/hello
    Hello, World!

## Supported Requests

The combination of the processor and the HTTP target supports the following API:

### GET /help

Return help about the supported API.

### GET /hello

Return "Hello, World!"

### GET /json

Return some valid JSON.

### POST /echo

Return back the data that was sent in the body of the POST. The processor
does not process the body, so an arbitrary amount of data may be streamed.

### GET /addHeader

Same as "/hello", but the processing server adds the header
"x-external-processor-status" to the response.

### POST /echohashstream

Echo back the request body, calculate the SHA-256 hash of each body
and print out each to the output of the processor.

The processor does this with the STREAMING body mode. This means that
the checksum will work regardless of the size of the request and response
bodies. However, we can't manipulate the headers after the body chunks
have been sent, so all that we can do is print them to standard output.

Try this with a big request, like a big file, to see this, and run the processor
with the "-d" (debug) flag enabled to see the messages coming in. For example:

    curl 0:10000/echohashstream -X POST -T <some big file> > /dev/null

### POST /echohashbuffered

This is the same as /echohashstream, except that it uses a "buffered" processing
mode. This way, the processor can add two parameters to the HTTP response that include
the hash of the request and response bodies. (When used with the supplied "httptarget",
since this uses the "/echo" endpoint, both should be the same.)

Since this mode uses Envoy's built-in buffering mechanism, Envoy will return a 413
error if the request body is larger than Envoy's (configurable) buffer size.

### POST /echohashbufferedpartial

This is the same as /echohashbuffered, but it uses the "partial buffering" scheme.
This way, there will never be an error if the request body is too large. However, if
the request body is larger than Envoy's buffer size, then the processor will continue
without calculating any hashes. The result is that bodies smaller than the buffer size
will have the additional response headers added, and larger bodies will not.

### POST /echoencode

Echo back the request body, but base64-encode the response. The processor does
this using STREAMING mode (and the golang base64 module also operates in streaming
mode) so it will work with very large bodies.

To verify:

    curl 0:10000/echoencode -d 'Base64 encode this' | base64 -d -

### POST /checkJson

Same as /echo, but if the Content-Type field on the request is set to
"application/json", then return an error if the posted content is not a
valid JSON message.

The golang JSON processor is not capable of streaming, so for this the processor
uses BUFFERED body mode. So, an error will be returned if you try to send a
request body that is too large.

For example:

    $ curl 0:10000/checkJson -H "Content-Type: application/json" -d 'This is not JSON'
    Invalid JSON

    $ curl 0:10000/checkJson -H "Content-Type: application/json" -d '{"isThisJson": true}'
    {"isThisJson": true}

### GET /getToPost

Turn an HTTP GET from the client into a POST to the /echo API on the server.
This will respond with a generic JSON message.

### GET /notfound

Just return a 404.
