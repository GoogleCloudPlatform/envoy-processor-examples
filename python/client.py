"""
# Header Manipulation - Example Client Stub
----
Client sends a stream of requests and Server responds with a stream of responses
"""

from __future__ import print_function


import logging

import google.protobuf  # .internal.well_known_types
import grpc

import service_pb2
import service_pb2_grpc

logger = logging.getLogger()
logger.setLevel("INFO")



def get_request(end_of_stream=False):
    request = service_pb2.ProcessingRequest(
        request_headers=service_pb2.HttpHeaders(
            headers=service_pb2.HeaderMap(
                headers=list(
                    [
                        service_pb2.HeaderValue(key="a1", value="b1"),
                        service_pb2.HeaderValue(key="a2", value="b2"),
                    ]
                )
            ),
            end_of_stream=end_of_stream,
        ),
    )
    return request


def new_stream_requests():
    request = service_pb2.ProcessingRequest(
        request_headers=service_pb2.HttpHeaders(
            headers=service_pb2.HeaderMap(
                headers=list(
                    [
                        service_pb2.HeaderValue(key="a1", value="b1"),
                        service_pb2.HeaderValue(key="a2", value="b2"),
                    ]
                )
            ),
            end_of_stream=False,
        ),
    )
    yield request
    response = service_pb2.ProcessingRequest(
        response_headers=service_pb2.HttpHeaders(
            headers=service_pb2.HeaderMap(
                headers=list(
                    [
                        service_pb2.HeaderValue(key="r1", value="b1"),
                        service_pb2.HeaderValue(key="r2", value="b2"),
                    ]
                )
            ),
            end_of_stream=False,
        ),
    )
    yield response



def run():
    with grpc.insecure_channel("localhost:8080") as channel:
        # sent client request to server
        stub = service_pb2_grpc.ExternalProcessorStub(channel)
        # Test 1
        for response in stub.Process(new_stream_requests()):
            print(response)


if __name__ == "__main__":
    print("==" * 45)
    run()
    print("==" * 45)
