use anyhow::anyhow;
use async_trait::async_trait;
use url::Url;

use crate::traits::Connect;
use crate::traits::Proxy;
use crate::wrapper::WrapRW;

#[derive(Debug)]
pub(crate) struct Stdio {}

impl TryFrom<&Url> for Stdio {
    type Error = anyhow::Error;

    fn try_from(url: &Url) -> Result<Self, Self::Error> {
        if url.scheme() == "stdio" {
            Ok(Stdio {})
        } else {
            Err(anyhow!("Invalid URL: {}", url.as_str()))
        }
    }
}

#[async_trait]
impl Connect for Stdio {
    async fn connect(&mut self) -> anyhow::Result<Proxy> {
        Ok(WrapRW::new_boxed(tokio::io::stdin(), tokio::io::stdout()))
    }
}
