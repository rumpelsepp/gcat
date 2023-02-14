use async_trait::async_trait;
use tokio::io::{AsyncRead, AsyncWrite};

pub(crate) trait ProxyTrait: AsyncRead + AsyncWrite + Unpin + Send {}

pub(crate) type Proxy = Box<dyn ProxyTrait>;

#[async_trait]
pub(crate) trait Connect {
    async fn connect(&mut self) -> anyhow::Result<Proxy>;
}
