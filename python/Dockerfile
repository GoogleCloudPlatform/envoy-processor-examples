FROM marketplace.gcr.io/google/debian11
WORKDIR /home/callouts
COPY * ./
RUN apt-get update && apt-get install -y python3-grpc-tools
EXPOSE 50051
ENTRYPOINT ["python3 server.py"]
