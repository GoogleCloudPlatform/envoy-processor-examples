use bytes::{BufMut, BytesMut};

pub struct Encoder {
  leftovers: BytesMut,
}

impl Encoder {
  pub fn new() -> Self {
    Encoder {
      leftovers: BytesMut::new(),
    }
  }

  pub fn encode(&mut self, bytes: &[u8], last_chunk: bool) -> String {
    let mut enc_buf = BytesMut::new();
    // Take leftover bytes
    enc_buf.put(self.leftovers.split());
    enc_buf.put(bytes);
    if !last_chunk {
      let remaining = enc_buf.len() % 3;
      if remaining > 0 {
        let tmp = enc_buf.split_to(enc_buf.len() - remaining);
        self.leftovers = enc_buf;
        enc_buf = tmp;
      }
    }
    base64::encode(enc_buf)
  }
}

#[cfg(test)]
mod tests {
  use super::*;

  #[test]
  fn test_encode_even() {
    let mut encoder = Encoder::new();
    let result = encoder.encode(b"abc", true);
    assert_eq!(result, "YWJj");
  }

  #[test]
  fn test_encode_odd() {
    let mut encoder = Encoder::new();
    let result = encoder.encode(b"abcd", true);
    assert_eq!(result, "YWJjZA==");
  }

  #[test]
  fn test_encode_even_chunks() {
    let mut encoder = Encoder::new();
    assert_eq!(encoder.encode(b"abc", false), "YWJj");
    assert_eq!(encoder.encode(b"de", true), "ZGU=");
  }

  #[test]
  fn test_encode_odd_chunks() {
    let mut encoder = Encoder::new();
    assert_eq!(encoder.encode(b"abcde", false), "YWJj");
    assert_eq!(encoder.encode(b"fg", true), "ZGVmZw==");
  }
}
