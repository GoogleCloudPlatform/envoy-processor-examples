fn main() -> Result<(), Box<dyn std::error::Error>> {
  tonic_build::configure()
    .build_client(false)
    .build_server(true)
    .compile(
      &["../data-plane-api/envoy/service/ext_proc/v3/external_processor.proto"],
      &["../udpa", "../protoc-gen-validate", "../xds", "../data-plane-api"],
    )?;
  Ok(())
}
