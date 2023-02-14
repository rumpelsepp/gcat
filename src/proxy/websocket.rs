use std::io;
use std::pin::Pin;
use std::task::{Context, Poll};

use anyhow::anyhow;
use async_trait::async_trait;
use futures_util::{SinkExt, StreamExt};
use tokio::io::{AsyncRead, AsyncWrite};
use tokio_tungstenite as websocket;
use tungstenite::protocol::Message;
use url::Url;

use crate::traits::{Connect, Proxy};
use crate::wrapper::Wrap;

#[derive(Debug)]
pub(crate) struct WsClient {
    target: String,
}

#[derive(Debug)]
struct WsWrapper<T> {
    inner: websocket::WebSocketStream<T>,
}

impl<T: AsyncRead + AsyncWrite + Unpin> AsyncWrite for WsWrapper<T> {
    fn poll_write(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<io::Result<usize>> {
        let msg = Message::binary(buf);

        match self.inner.poll_ready_unpin(cx) {
            Poll::Pending => return Poll::Pending,
            Poll::Ready(res) => match res {
                Ok(()) => (),
                Err(e) => {
                    return Poll::Ready(Err(io::Error::new(io::ErrorKind::Other, e.to_string())))
                }
            },
        };

        match self.inner.start_send_unpin(msg) {
            Ok(_) => Poll::Ready(Ok(buf.len())),
            Err(e) => Poll::Ready(Err(io::Error::new(io::ErrorKind::Other, e.to_string()))),
        }
    }

    fn poll_flush(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Result<(), io::Error>> {
        self.inner
            .poll_flush_unpin(cx)
            .map_err(|e| io::Error::new(io::ErrorKind::Other, e.to_string()))
    }

    fn poll_shutdown(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
    ) -> Poll<Result<(), io::Error>> {
        self.inner
            .poll_close_unpin(cx)
            .map_err(|e| io::Error::new(io::ErrorKind::Other, e.to_string()))
    }
}

impl<T: AsyncRead + AsyncWrite + Unpin> AsyncRead for WsWrapper<T> {
    fn poll_read(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut tokio::io::ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        match self.inner.poll_next_unpin(cx) {
            Poll::Pending => Poll::Pending,
            Poll::Ready(v) => match v {
                None => Poll::Ready(Ok(())),
                Some(item) => match item {
                    Ok(msg) => match msg {
                        Message::Binary(data) => {
                            buf.put_slice(&data);
                            Poll::Ready(Ok(()))
                        }
                        msgtype => Poll::Ready(Err(io::Error::new(
                            io::ErrorKind::Other,
                            format!("unsupported message type: {:?}", msgtype),
                        ))),
                    },
                    Err(e) => Poll::Ready(Err(io::Error::new(io::ErrorKind::Other, e.to_string()))),
                },
            },
        }
    }
}

impl TryFrom<&Url> for WsClient {
    type Error = anyhow::Error;

    fn try_from(url: &Url) -> Result<Self, Self::Error> {
        if (url.scheme() == "ws" || url.scheme() == "wss") && url.has_host() {
            return Ok(Self {
                target: url.to_string(),
            });
        }

        Err(anyhow!("Invalid URL: {}", url.as_str()))
    }
}

#[async_trait]
impl Connect for WsClient {
    async fn connect(&mut self) -> anyhow::Result<Proxy> {
        let (stream, _) = websocket::connect_async(&self.target).await?;
        Ok(Wrap::new_boxed(WsWrapper { inner: stream }))
    }
}
