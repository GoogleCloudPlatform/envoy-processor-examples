use {
  futures::Stream,
  crate::pb::{external_processor_server::ExternalProcessor, ProcessingRequest, ProcessingResponse},
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
    unimplemented!()
  }
}
