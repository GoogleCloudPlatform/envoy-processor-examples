use {
  envoy_control_plane::envoy::{
    config::core::v3::HeaderMap,
    r#type::v3::HttpStatus,
    service::ext_proc::v3::{
      external_processor_server::ExternalProcessor, processing_request, processing_response,
      ImmediateResponse, ProcessingRequest, ProcessingResponse,
    },
  },
  futures::{channel::mpsc::UnboundedSender, Stream},
  std::pin::Pin,
  tonic::{Request, Response, Status, Streaming},
};

type ExternalProcessorStream =
  Pin<Box<dyn Stream<Item = Result<ProcessingResponse, Status>> + Send>>;

pub struct ExampleProcessor {}

#[tonic::async_trait]
impl ExternalProcessor for ExampleProcessor {
  type ProcessStream = ExternalProcessorStream;

  async fn process(
    &self,
    request: Request<Streaming<ProcessingRequest>>,
  ) -> Result<Response<ExternalProcessorStream>, Status> {
    let mut req = request.into_inner();
    if let Some(first_msg) = req.message().await? {
      if let Some(processing_request::Request::RequestHeaders(req_headers)) = first_msg.request {
        let (sender, receiver) = futures::channel::mpsc::unbounded();
        let path = get_header_value(&req_headers.headers, ":path");
        match path {
          Some("/notfound") => handle_not_found(sender)?,
          _ => sender.close_channel(),
        }
        return Ok(Response::new(Box::pin(receiver)));
      }
    }
    // By default, just close the stream.
    Ok(Response::new(Box::pin(futures::stream::empty())))
  }
}

// Handle a not found by immediately writing to the channel.
fn handle_not_found(
  sender: UnboundedSender<Result<ProcessingResponse, Status>>,
) -> Result<(), Status> {
  if let Err(_) = sender.unbounded_send(Ok(new_error(404, "not found"))) {
    return Err(Status::internal("stream error"));
  }
  Ok(())
}

fn new_error(status: i32, msg: &str) -> ProcessingResponse {
  let immediate = ImmediateResponse {
    status: Some(HttpStatus { code: status }),
    headers: None,
    body: String::from(""),
    details: String::from(msg),
    grpc_status: None,
  };
  ProcessingResponse {
    response: Some(processing_response::Response::ImmediateResponse(immediate)),
    mode_override: None,
    dynamic_metadata: None,
  }
}

fn get_header_value<'a>(header_map: &'a Option<HeaderMap>, name: &str) -> Option<&'a str> {
  match header_map {
    Some(headers) => {
      for header in &headers.headers {
        if header.key == name {
          return Some(&header.value);
        }
      }
      None
    }
    None => None,
  }
}
