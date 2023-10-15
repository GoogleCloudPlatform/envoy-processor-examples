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
# A simple client to test the python ext-proc server.
"""

from __future__ import print_function


import logging

import google.protobuf  # .internal.well_known_types
import grpc

import service_pb2
import service_pb2_grpc

logger = logging.getLogger()
logger.setLevel("INFO")

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
        stub = service_pb2_grpc.ExternalProcessorStub(channel)
        for response in stub.Process(new_stream_requests()):
            print(response)


if __name__ == "__main__":
    print("==" * 45)
    run()
    print("==" * 45)
