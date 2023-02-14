use std::net::Ipv4Addr;

use anyhow::{anyhow, bail, Result};
use async_trait::async_trait;
use log::*;
use serde::Deserialize;
use url::Url;

use crate::parse_qs;
use crate::traits::{Connect, Proxy};
use crate::wrapper::Wrap;

const DEFAULT_DEV: &str = "tun%d";
const DEFAULT_MTU: i32 = 1500;

fn get_default_dev() -> String {
    DEFAULT_DEV.into()
}

fn get_default_mtu() -> i32 {
    DEFAULT_MTU
}

#[derive(Debug, Deserialize)]
struct TunSettings {
    #[serde(default = "get_default_dev")]
    dev: String,
    #[serde(default = "get_default_mtu")]
    mtu: i32,
}

impl Default for TunSettings {
    fn default() -> Self {
        Self {
            dev: DEFAULT_DEV.into(),
            mtu: DEFAULT_MTU,
        }
    }
}

pub(crate) struct TunDevice {
    config: tun::Configuration,
}

#[async_trait]
impl Connect for TunDevice {
    async fn connect(&mut self) -> anyhow::Result<Proxy> {
        Ok(Wrap::new_boxed(tun::create_as_async(&self.config)?))
    }
}

fn prefix_to_mask(prefix: u8) -> Result<(u8, u8, u8, u8)> {
    if prefix > 32 {
        bail!("prefix must not be > 32");
    }

    let mut netmask: [u8; 4] = [255, 255, 255, 255];
    for i in 0..=3 {
        let seen_bits = (i + 1) * 8;

        if seen_bits > prefix {
            let overlap: i8 = (prefix as i8 - seen_bits as i8).abs();
            let bitmask = if overlap >= 8 {
                0x00
            } else {
                let mut out: u8 = 0x00;
                for bit_pos in 0..overlap {
                    out |= 1 << bit_pos;
                }
                !out
            };
            netmask[i as usize] &= bitmask;
        }
    }

    Ok((netmask[0], netmask[1], netmask[2], netmask[3]))
}

impl TryFrom<&Url> for TunDevice {
    type Error = anyhow::Error;

    fn try_from(url: &Url) -> Result<Self, Self::Error> {
        if url.scheme() == "tun" && url.has_host() && url.path() != "" {
            let host = url.host_str().ok_or_else(|| anyhow!("no host"))?;
            let ip: Ipv4Addr = host.parse()?;
            let raw_path = url.path();
            let mask = prefix_to_mask(raw_path.strip_prefix('/').unwrap_or(raw_path).parse()?)?;

            let config: TunSettings = parse_qs!(url).unwrap_or_else(TunSettings::default);

            let mut tun_config = tun::Configuration::default();
            tun_config
                .address(ip)
                .netmask(mask)
                .mtu(config.mtu)
                .name(&config.dev)
                .up();

            debug!("{:#?}", tun_config);

            return Ok(Self { config: tun_config });
        }

        Err(anyhow!("Invalid URL: {}", url.as_str()))
    }
}
