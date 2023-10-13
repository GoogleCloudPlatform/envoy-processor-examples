"""
# Header Manipulation - Example Server
----
Client sends a stream of requests and Server responds with a stream of responses
"""
from concurrent import futures
import logging

import grpc

import service_pb2
import service_pb2_grpc
from typing import Iterator
from grpc import ServicerContext

logger = logging.getLogger()
logger.setLevel("INFO")


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
            response = get_response()
            yield response


def serve():
    port = "50051"
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    service_pb2_grpc.add_ExternalProcessorServicer_to_server(
        TrafficExtensionCallout(), server
    )
    server.add_insecure_port("[::]:" + port)
    server.start()
    print("Server started, listening on " + port)
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig()
    serve()
