use {
  crate::base64::Encoder,
  envoy_control_plane::envoy::{
    config::core::v3::{HeaderMap, HeaderValue, HeaderValueOption},
    extensions::filters::http::ext_proc::v3::{processing_mode, ProcessingMode},
    r#type::v3::HttpStatus,
    service::ext_proc::v3::{
      body_mutation, external_processor_server::ExternalProcessor, processing_request,
      processing_response, BodyMutation, BodyResponse, CommonResponse, HeaderMutation,
      HeadersResponse, HttpBody, HttpHeaders, ImmediateResponse, ProcessingRequest,
      ProcessingResponse,
    },
  },
  futures::{channel::mpsc::UnboundedSender, SinkExt, Stream},
  std::{pin::Pin, str},
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
    let mut stream = request.into_inner();
    if let Some(req_headers) = get_request_headers(&mut stream).await {
      let (sender, receiver) = futures::channel::mpsc::unbounded();
      let path = get_header_value(&req_headers.headers, ":path");
      match path {
        Some("/notfound") => handle_not_found(sender)?,
        Some("/addHeader") => {
          tokio::task::spawn(async move {
            handle_add_header(sender, stream).await;
          });
        }
        Some("/checkJson") => {
          tokio::task::spawn(async move {
            handle_check_json(&req_headers, sender, stream).await;
          });
        }
        Some("/echoencode") => {
          tokio::task::spawn(async move {
            handle_echo_encode(sender, stream).await;
          });
        }
        _ => sender.close_channel(),
      }
      return Ok(Response::new(Box::pin(receiver)));
    }
    // By default, just close the stream.
    Ok(Response::new(Box::pin(futures::stream::empty())))
  }
}

// Handle a not found by immediately writing to the channel and letting
// it close.
fn handle_not_found(
  sender: UnboundedSender<Result<ProcessingResponse, Status>>,
) -> Result<(), Status> {
  if sender
    .unbounded_send(Ok(new_error(404, "not found")))
    .is_err()
  {
    return Err(Status::internal("stream error"));
  }
  Ok(())
}

// Add a header to the response.
async fn handle_add_header(
  mut sender: UnboundedSender<Result<ProcessingResponse, Status>>,
  mut stream: Streaming<ProcessingRequest>,
) {
  // Send back a response that changes the request header for the HTTP target.
  let mut req_headers_cr = CommonResponse::default();
  add_set_header(&mut req_headers_cr, ":path", "/hello");
  let req_headers_resp = ProcessingResponse {
    response: Some(processing_response::Response::RequestHeaders(
      HeadersResponse {
        response: Some(req_headers_cr),
      },
    )),
    ..Default::default()
  };
  sender.send(Ok(req_headers_resp)).await.ok();

  if get_response_headers(&mut stream).await.is_some() {
    let mut resp_headers_cr = CommonResponse::default();
    add_set_header(
      &mut resp_headers_cr,
      "x-external-processor-status",
      "Powered by Rust!",
    );
    let resp_headers_resp = ProcessingResponse {
      response: Some(processing_response::Response::ResponseHeaders(
        HeadersResponse {
          response: Some(resp_headers_cr),
        },
      )),
      ..Default::default()
    };
    sender.send(Ok(resp_headers_resp)).await.ok();
  }
  // Fall through if we get the wrong message.
}

// Check that the request body is JSON, and if so, reject the request
// if it is invalid and add a header to the response otherwise.
async fn handle_check_json(
  request_headers: &HttpHeaders,
  mut sender: UnboundedSender<Result<ProcessingResponse, Status>>,
  mut stream: Streaming<ProcessingRequest>,
) {
  let is_json = matches!(
    get_header_value(&request_headers.headers, "content-type"),
    Some("application/json")
  );
  let mut req_headers_cr = CommonResponse::default();
  add_set_header(&mut req_headers_cr, ":path", "/echo");
  let mut req_headers_resp = ProcessingResponse {
    response: Some(processing_response::Response::RequestHeaders(
      HeadersResponse {
        response: Some(req_headers_cr),
      },
    )),
    ..Default::default()
  };
  if is_json {
    // Switch to a mode in which we get the body only if it's JSON.
    req_headers_resp.mode_override = Some(ProcessingMode {
      request_body_mode: processing_mode::BodySendMode::Buffered as i32,
      ..Default::default()
    });
  }
  sender.send(Ok(req_headers_resp)).await.ok();

  let mut json_status = "Not JSON";

  if is_json {
    if let Some(request_body) = get_request_body(&mut stream).await {
      if let Ok(body_str) = str::from_utf8(&request_body.body) {
        match json::parse(body_str) {
          Ok(_) => json_status = "Valid JSON",
          Err(_) => {
            sender.send(Ok(new_error(400, "Invalid JSON"))).await.ok();
            return;
          }
        }
      }
    }
    let req_body_response = ProcessingResponse {
      response: Some(processing_response::Response::RequestBody(
        BodyResponse::default(),
      )),
      ..Default::default()
    };
    sender.send(Ok(req_body_response)).await.ok();
  }

  if get_response_headers(&mut stream).await.is_some() {
    let mut resp_headers_cr = CommonResponse::default();
    add_set_header(&mut resp_headers_cr, "x-json-status", json_status);
    let resp_headers_resp = ProcessingResponse {
      response: Some(processing_response::Response::ResponseHeaders(
        HeadersResponse {
          response: Some(resp_headers_cr),
        },
      )),
      ..Default::default()
    };
    sender.send(Ok(resp_headers_resp)).await.ok();
  }
  // Fall through if we get the wrong message.
}

// Encode the response in a streaming way into base64.
async fn handle_echo_encode(
  mut sender: UnboundedSender<Result<ProcessingResponse, Status>>,
  mut stream: Streaming<ProcessingRequest>,
) {
  // Send back a response that changes the request URL for the HTTP target.
  let mut req_headers_cr = CommonResponse::default();
  add_set_header(&mut req_headers_cr, ":path", "/echo");
  let req_headers_resp = ProcessingResponse {
    response: Some(processing_response::Response::RequestHeaders(
      HeadersResponse {
        response: Some(req_headers_cr),
      },
    )),
    mode_override: Some(ProcessingMode {
      response_body_mode: processing_mode::BodySendMode::Streamed as i32,
      ..Default::default()
    }),
    ..Default::default()
  };
  sender.send(Ok(req_headers_resp)).await.ok();

  let mut encoder = Encoder::new();

  // Loop to process messages, because we act on both response headers
  // and also on each chunk of the body.
  while let Ok(Some(next_msg)) = stream.message().await {
    match next_msg.request {
      Some(processing_request::Request::ResponseHeaders(_)) => {
        let resp_headers_resp = ProcessingResponse {
          // Be sure to change the path so that the HTTP target works,
          // and clear content-length since it will change as we encode.
          response: Some(processing_response::Response::ResponseHeaders(
            HeadersResponse {
              response: Some(CommonResponse {
                header_mutation: Some(HeaderMutation {
                  set_headers: vec![HeaderValueOption {
                    header: Some(HeaderValue {
                      key: ":path".into(),
                      value: "/echo".into(),
                    }),
                    ..Default::default()
                  }],
                  remove_headers: vec!["content-length".into()],
                }),
                ..Default::default()
              }),
            },
          )),
          ..Default::default()
        };
        sender.send(Ok(resp_headers_resp)).await.ok();
      }
      Some(processing_request::Request::ResponseBody(chunk)) => {
        let new_body = encoder.encode(&chunk.body, chunk.end_of_stream);
        let resp_body_resp = ProcessingResponse {
          response: Some(processing_response::Response::ResponseBody(BodyResponse {
            response: Some(CommonResponse {
              body_mutation: Some(BodyMutation {
                mutation: Some(body_mutation::Mutation::Body(new_body.into())),
              }),
              ..Default::default()
            }),
          })),
          ..Default::default()
        };
        sender.send(Ok(resp_body_resp)).await.ok();
      }
      _ => {}
    }
  }
}

async fn get_request_headers(stream: &mut Streaming<ProcessingRequest>) -> Option<HttpHeaders> {
  if let Ok(Some(next_msg)) = stream.message().await {
    if let Some(processing_request::Request::RequestHeaders(hdrs)) = next_msg.request {
      return Some(hdrs);
    }
  }
  None
}

async fn get_request_body(stream: &mut Streaming<ProcessingRequest>) -> Option<HttpBody> {
  if let Ok(Some(next_msg)) = stream.message().await {
    if let Some(processing_request::Request::RequestBody(hdrs)) = next_msg.request {
      return Some(hdrs);
    }
  }
  None
}

async fn get_response_headers(stream: &mut Streaming<ProcessingRequest>) -> Option<HttpHeaders> {
  if let Ok(Some(next_msg)) = stream.message().await {
    if let Some(processing_request::Request::ResponseHeaders(hdrs)) = next_msg.request {
      return Some(hdrs);
    }
  }
  None
}

fn new_error(status: i32, msg: &str) -> ProcessingResponse {
  ProcessingResponse {
    response: Some(processing_response::Response::ImmediateResponse(
      ImmediateResponse {
        status: Some(HttpStatus { code: status }),
        details: msg.into(),
        ..Default::default()
      },
    )),
    ..Default::default()
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

fn add_set_header(cr: &mut CommonResponse, key: &str, value: &str) {
  let new_header = HeaderValueOption {
    header: Some(HeaderValue {
      key: key.into(),
      value: value.into(),
    }),
    ..Default::default()
  };
  match &mut cr.header_mutation {
    Some(hm) => hm.set_headers.push(new_header),
    None => {
      let mut new_hm = HeaderMutation::default();
      new_hm.set_headers.push(new_header);
      cr.header_mutation = Some(new_hm);
    }
  }
}
