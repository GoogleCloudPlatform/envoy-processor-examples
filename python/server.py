# Copyright 2023 Google LLC.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
# Example external processing server
----
This server does two things:
 When it gets request_headers, it replaces the Host header with service-extensions.com
  and resets the path to /.
 When it gets response_headers, it adds a "hello: service-extensions" response header.
"""
from concurrent import futures
from http.server import BaseHTTPRequestHandler, HTTPServer
import logging

import grpc

import service_pb2
import service_pb2_grpc

import _credentials

from typing import Iterator, List
from grpc import ServicerContext

logger = logging.getLogger()
logger.setLevel("INFO")

# Backend services on GCE VMs, GKE and hybrid use this port.
EXT_PROC_SECURE_PORT = 443
# Backend services on Cloud Run use this port.
EXT_PROC_INSECURE_PORT = 8080
# Cloud health checks use this port.
HEALTH_CHECK_PORT = 80

# Returns an ext_proc HeadersResponse for adding a list of headers.
# clear_route_cache should be set to influence service selection for route
# extensions.
def add_headers_mutation(headers: List[tuple[str, str]], clear_route_cache: bool = False) -> service_pb2.HeadersResponse:
    response_header_mutation = service_pb2.HeadersResponse()
    response_header_mutation.response.header_mutation.set_headers.extend([
        service_pb2.HeaderValueOption(
            header=service_pb2.HeaderValue(key=k, raw_value=bytes(v, "utf-8"))
        ) for k, v in headers
    ])
    if clear_route_cache:
        response_header_mutation.response.clear_route_cache = True
    return response_header_mutation


class CalloutProcessor(service_pb2_grpc.ExternalProcessorServicer):
    def Process(
        self,
        request_iterator: Iterator[service_pb2.ProcessingRequest],
        context: ServicerContext,
    ) -> Iterator[service_pb2.ProcessingResponse]:
        for request in request_iterator:
            print(request)
            if request.HasField("response_headers"):
                response_header_mutation = add_headers_mutation([("hello", "service-extensions")])
                yield service_pb2.ProcessingResponse(response_headers=response_header_mutation)
            elif request.HasField("request_headers"):
                request_header_mutation = add_headers_mutation([
                    ("host", "service-extensions.com"),
                    (":path", "/")
                ], clear_route_cache=True)
                yield service_pb2.ProcessingResponse(request_headers=request_header_mutation)

class HealthCheckServer(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()


def serve():
    health_server = HTTPServer(("0.0.0.0", HEALTH_CHECK_PORT), HealthCheckServer)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    service_pb2_grpc.add_ExternalProcessorServicer_to_server(
        CalloutProcessor(), server
    )
    server_credentials = grpc.ssl_server_credentials(
        (
            (
                _credentials.SERVER_CERTIFICATE_KEY,
                _credentials.SERVER_CERTIFICATE,
            ),
        )
    )
    server.add_secure_port("0.0.0.0:%d" % EXT_PROC_SECURE_PORT, server_credentials)
    server.add_insecure_port("0.0.0.0:%d" % EXT_PROC_INSECURE_PORT)
    server.start()
    print("Server started, listening on %d and %d" % (EXT_PROC_SECURE_PORT, EXT_PROC_INSECURE_PORT))
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
