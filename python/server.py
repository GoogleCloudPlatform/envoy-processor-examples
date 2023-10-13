"""
# Header Manipulation - Example Server
----
Client sends a stream of requests and Server responds with a stream of responses
"""
from concurrent import futures
from http.server import BaseHTTPRequestHandler, HTTPServer
import logging

import grpc

import service_pb2
import service_pb2_grpc

import _credentials

from typing import Iterator
from grpc import ServicerContext

logger = logging.getLogger()
logger.setLevel("INFO")

EXT_PROC_PORT = 443
HEALTH_CHECK_PORT = 80

def get_response():
    header_mutation = service_pb2.HeaderMutation(
        set_headers=list(
            [
                service_pb2.HeaderValueOption(
                    header=service_pb2.HeaderValue(key="hello", value="ext-proc")
                ),
            ]
        ),
    )
    request_headers = service_pb2.HeadersResponse(
        response=service_pb2.CommonResponse(
            header_mutation=header_mutation,
        )
    )
    response = service_pb2.ProcessingResponse(request_headers=request_headers)
    return response


class TrafficExtensionCallout(service_pb2_grpc.ExternalProcessorServicer):
    def ProcessSimple(self, request: service_pb2.ProcessingRequest, context):
        print(request)
        response = get_response()
        return response

    def Process(
        self,
        request_iterator: Iterator[service_pb2.ProcessingRequest],
        context: ServicerContext,
    ) -> Iterator[service_pb2.ProcessingResponse]:
        for request in request_iterator:
            print(request)
            yield get_response()


class HealthCheckServer(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()


def serve():
    health_server = HTTPServer(("0.0.0.0", HEALTH_CHECK_PORT), HealthCheckServer)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    service_pb2_grpc.add_ExternalProcessorServicer_to_server(
        TrafficExtensionCallout(), server
    )
    server_credentials = grpc.ssl_server_credentials(
        (
            (
                _credentials.SERVER_CERTIFICATE_KEY,
                _credentials.SERVER_CERTIFICATE,
            ),
        )
    )
    server.add_secure_port("0.0.0.0:%d" % EXT_PROC_PORT, server_credentials)
    server.start()
    print("Server started, listening on %d" % EXT_PROC_PORT)
    try:
        health_server.serve_forever()
    except KeyboardInterrupt:
        print("Server interrupted")
    finally:
        server.stop()
        health_server.server_close()


if __name__ == "__main__":
    logging.basicConfig()
    serve()
