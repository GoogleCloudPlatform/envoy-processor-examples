pub mod pb {
  tonic::include_proto!("envoy.service.ext_proc.v3");
}

use {crate::service::ExampleProcessor, tonic::transport::Server};

mod service;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
  let addr = "127.0.0.1:11111".parse().unwrap();
  println!("Server listening on {}", addr);
  let server = ExampleProcessor {};
  Server::builder()
    .add_service(pb::external_processor_server::ExternalProcessorServer::new(
      server,
    ))
    .serve(addr)
    .await?;
  Ok(())
}
