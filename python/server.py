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
# Header Manipulation - Example Server

Description: When a client sends a stream of gRPC requests, the server responds
with a stream of responses. In this example, the server responds to the client
with an example header to add to the request. The header will be named "hello"
and will have the value "ext-proc".
"""
from concurrent import futures

# [START serviceextension_add_header]
import grpc
import service_pb2
import service_pb2_grpc
from typing import Iterator


class CalloutExample(service_pb2_grpc.ExternalProcessorServicer):
    def Process(
        self,
        request_iterator: Iterator[service_pb2.ProcessingRequest],
        context: grpc.ServicerContext,
    ) -> Iterator[service_pb2.ProcessingResponse]:
        for request in request_iterator:
            # TODO(Developer): Process the input request & prepare a response
            # Add header `hello: ext-proc`
            header_mutation = service_pb2.HeaderMutation(
                set_headers=list(
                    [
                        service_pb2.HeaderValueOption(
                            header=service_pb2.HeaderValue(
                                key="hello", value="ext-proc"
                            )
                        )
                    ]
                )
            )
            response = service_pb2.ProcessingResponse(
                request_headers=service_pb2.HeadersResponse(
                    response=service_pb2.CommonResponse(
                        status=service_pb2.CommonResponse.ResponseStatus.CONTINUE_AND_REPLACE,
                        header_mutation=header_mutation,
                    )
                )
            )
            # Client is expected to send a stream of request
            yield response


# [END serviceextension_add_header]
def serve(port="50051"):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    service_pb2_grpc.add_ExternalProcessorServicer_to_server(CalloutExample(), server)
    server.add_insecure_port("[::]:" + port)
    server.start()
    print("Server started, listening on " + port)
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
