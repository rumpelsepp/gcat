#[macro_export]
macro_rules! parse_qs {
    ($url:ident) => {
        match $url.query() {
            Some(u) => Some(serde_qs::from_str(u)?),
            None => None,
        }
    };
}
