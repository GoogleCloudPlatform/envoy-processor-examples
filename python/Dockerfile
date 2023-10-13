FROM marketplace.gcr.io/google/debian11
WORKDIR /home/callouts
COPY * ./
COPY credentials/ ./credentials
RUN apt-get update && apt-get install -y python3-grpc-tools
EXPOSE 80
EXPOSE 50051
CMD [ "/usr/bin/python3", "server.py" ]
