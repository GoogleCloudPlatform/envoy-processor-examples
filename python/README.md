This folder contains a simple Python gRPC server generated using the external processor `service.proto`.
The gRPC server adds a header `hello: ext-proc` to a processing request. It is meant to be used with a
load balancer service extension configured to call the external processing backend service for REQUEST_HEADERS event.
