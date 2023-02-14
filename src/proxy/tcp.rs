use std::time::Duration;

use anyhow::anyhow;
use async_trait::async_trait;
use serde::Deserialize;
use tokio::net::{TcpListener, TcpStream};
use url::Url;

use crate::parse_qs;
use crate::traits::Connect;
use crate::traits::Proxy;
use crate::wrapper::Wrap;

#[derive(Debug, Deserialize)]
struct TcpSettings {
    #[serde(default)]
    nodelay: bool,
    #[serde(with = "humantime_serde::option")]
    linger: Option<Duration>,
    ttl: Option<u32>,
}

macro_rules! apply_settings {
    ($self:ident, $stream:ident) => {
        if let Some(settings) = &$self.settings {
            $stream.set_nodelay(settings.nodelay)?;

            if let Some(ttl) = settings.ttl {
                $stream.set_ttl(ttl)?;
            }

            $stream.set_linger(settings.linger)?;
        }
    };
}

#[derive(Debug)]
pub(crate) struct TcpClient {
    target: (String, u16),
    settings: Option<TcpSettings>,
}

impl TryFrom<&Url> for TcpClient {
    type Error = anyhow::Error;

    fn try_from(url: &Url) -> Result<Self, Self::Error> {
        if url.scheme() == "tcp" && url.has_host() {
            let host = url.host_str().unwrap();
            let port = url.port().unwrap();
            let settings = parse_qs!(url);
            return Ok(Self {
                target: (host.into(), port),
                settings,
            });
        }

        Err(anyhow!("Invalid URL: {}", url.as_str()))
    }
}

#[async_trait]
impl Connect for TcpClient {
    async fn connect(&mut self) -> anyhow::Result<Proxy> {
        let stream = TcpStream::connect(&self.target).await?;

        apply_settings!(self, stream);

        Ok(Wrap::new_boxed(stream))
    }
}

#[derive(Debug)]
pub(crate) struct TcpServer {
    target: (String, u16),
    listener: Option<TcpListener>,
    settings: Option<TcpSettings>,
}

impl TryFrom<&Url> for TcpServer {
    type Error = anyhow::Error;

    fn try_from(url: &Url) -> Result<Self, Self::Error> {
        if url.scheme() == "tcp-server" && url.has_host() {
            let host = url.host_str().ok_or_else(|| anyhow!("no host"))?;
            let port = url.port().ok_or_else(|| anyhow!("no port"))?;
            let settings = parse_qs!(url);

            return Ok(Self {
                target: (host.into(), port),
                listener: None,
                settings,
            });
        }

        Err(anyhow!("Invalid URL: {}", url.as_str()))
    }
}

#[async_trait]
impl Connect for TcpServer {
    async fn connect(&mut self) -> anyhow::Result<Proxy> {
        if self.listener.is_none() {
            self.listener = Some(TcpListener::bind(&self.target).await?);
        }

        let listener = self.listener.as_ref().unwrap();
        let (stream, _) = listener.accept().await?;

        apply_settings!(self, stream);

        Ok(Wrap::new_boxed(stream))
    }
}
