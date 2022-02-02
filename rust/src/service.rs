use {
  envoy_control_plane::envoy::service::ext_proc::v3::{
    external_processor_server::ExternalProcessor, ProcessingRequest, ProcessingResponse,
  },
  futures::Stream,
  std::pin::Pin,
  tonic::{Request, Response, Status, Streaming},
};

pub struct ExampleProcessor {}

#[tonic::async_trait]
impl ExternalProcessor for ExampleProcessor {
  type ProcessStream = Pin<Box<dyn Stream<Item = Result<ProcessingResponse, Status>> + Send>>;

  async fn process(
    &self,
    request: Request<Streaming<ProcessingRequest>>,
  ) -> Result<Response<Self::ProcessStream>, Status> {
    let mut req = request.into_inner();
    if let Some(first_msg) = req.message().await? {
      // TODO check the type and the path
      // Depending on the path, dispatch to various other things.
      unimplemented!()
    }
    // End of stream
    Ok(Response::new(Box::pin(futures::stream::empty())))
  }
}
