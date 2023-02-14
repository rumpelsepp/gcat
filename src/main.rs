use anyhow::anyhow;
use clap::Parser;
use log::*;
use tokio;
use tokio::io::copy_bidirectional;
use url::Url;

mod macros;
mod proxy;
mod traits;
mod wrapper;

use crate::traits::{Connect, Proxy};

#[derive(Debug, Parser)]
#[clap(name = "rscat")]
#[clap(author, version, about, long_about = None)]
struct Cli {
    left: Url,
    right: Url,

    #[arg(short, long, conflicts_with = "loop")]
    concurrent: bool,

    #[arg(short, long, conflicts_with = "concurrent")]
    r#loop: bool,

    #[arg(short, long)]
    quiet: bool,

    #[arg(short, long, action = clap::ArgAction::Count)]
    verbose: u8,

    /// Timestamp (sec, ms, ns, none)
    #[arg(short, long)]
    timestamps: Option<stderrlog::Timestamp>,
}

async fn connect(url: &Url) -> anyhow::Result<Proxy> {
    let scheme = url.scheme();
    match scheme {
        "stdio" => proxy::stdio::Stdio::try_from(url)?.connect().await,
        "tcp" => proxy::tcp::TcpClient::try_from(url)?.connect().await,
        "tcp-server" => proxy::tcp::TcpServer::try_from(url)?.connect().await,
        "tun" => proxy::tun::TunDevice::try_from(url)?.connect().await,
        "quic" => proxy::quic::QuicClient::try_from(url)?.connect().await,
        "ws" | "wss" => proxy::websocket::WsClient::try_from(url)?.connect().await,
        _ => Err(anyhow!("unsupported scheme: '{}'", scheme)),
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Cli::parse();

    stderrlog::new()
        .module(module_path!())
        .quiet(args.quiet)
        .verbosity(args.verbose as usize)
        .timestamp(args.timestamps.unwrap_or(stderrlog::Timestamp::Off))
        .init()?;

    debug!("{:#?}", args.right);
    debug!("{:#?}", args.left);

    loop {
        if args.concurrent {
            let left_url = args.left.clone();
            let right_url = args.right.clone();
            tokio::spawn(async move {
                // TODO: remove unwrap
                let mut left_proxy = connect(&left_url).await.unwrap();
                let mut right_proxy = connect(&right_url).await.unwrap();
                copy_bidirectional(&mut left_proxy, &mut right_proxy)
                    .await
                    .unwrap();
            });
        } else {
            let mut left_proxy = connect(&args.left).await?;
            let mut right_proxy = connect(&args.right).await?;
            copy_bidirectional(&mut left_proxy, &mut right_proxy).await?;
        }

        if !args.concurrent || !args.r#loop {
            break;
        }
    }

    Ok(())
}
