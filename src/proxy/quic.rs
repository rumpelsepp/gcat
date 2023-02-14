use anyhow::{anyhow, bail};
use async_trait::async_trait;
use log::*;
use serde::Deserialize;
use tokio::net::lookup_host;
use url::Url;

use crate::parse_qs;
use crate::traits::Connect;
use crate::traits::Proxy;
use crate::wrapper::WrapRW;

#[derive(Debug)]
pub(crate) struct QuicClient {
    target: (String, u16),
    // settings: Option<TcpSettings>,
}

impl TryFrom<&Url> for QuicClient {
    type Error = anyhow::Error;

    fn try_from(url: &Url) -> Result<Self, Self::Error> {
        if url.scheme() == "quic" && url.has_host() {
            let host = url.host_str().unwrap();
            let port = url.port().unwrap();
            return Ok(Self {
                target: (host.into(), port),
            });
        }

        Err(anyhow!("Invalid URL: {}", url.as_str()))
    }
}

#[async_trait]
impl Connect for QuicClient {
    async fn connect(&mut self) -> anyhow::Result<Proxy> {
        let hosts = lookup_host(self.target.clone()).await?;
        let mut ep = quinn::Endpoint::client("[::]:0".parse().unwrap())?;
        let client_config = quinn::ClientConfig::with_native_roots();
        ep.set_default_client_config(client_config);

        debug!("resolved {:?}", self.target.clone());
        let mut conn: Option<quinn::Connection> = None;

        for host in hosts {
            debug!(" - {}", host);
            // TODO: add hostname instead of foo
            conn = match ep.connect(host, "foo")?.await {
                Ok(conn) => Some(conn),
                Err(e) => {
                    warn!("connection error: {e}");
                    continue;
                }
            }
        }

        let err_msg = format!("could not connect to: {:?}", self.target);
        let Some(conn) = conn else {
            bail!(err_msg);
        };

        let (w, r) = conn.open_bi().await?;

        Ok(WrapRW::new_boxed(r, w))
    }
}
