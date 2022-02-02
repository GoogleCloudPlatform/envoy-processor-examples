use {
  crate::service::ExampleProcessor,
  envoy_control_plane::envoy::service::ext_proc::v3::external_processor_server::ExternalProcessorServer,
  tonic::transport::Server,
};

mod service;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
  // TODO use a command-line processing library here.
  let addr = "127.0.0.1:10002".parse().unwrap();
  println!("Server listening on {}", addr);
  let server = ExampleProcessor {};
  Server::builder()
    .add_service(ExternalProcessorServer::new(server))
    .serve(addr)
    .await?;
  Ok(())
}
