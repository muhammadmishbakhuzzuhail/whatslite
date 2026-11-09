//! Raw byte-stream transport for mobile clients and independently keyed mocks.
//!
//! Framing and Noise remain in the protocol layer. This adapter neither adds
//! TLS nor performs an HTTP upgrade, even when the selected port is 443.

use std::sync::Arc;

use anyhow::{Context, bail};
use async_trait::async_trait;
use bytes::BytesMut;
use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};
use tokio::sync::{Mutex, watch};
use wacore::net::{Transport, TransportEvent, TransportFactory};

const READ_CAPACITY: usize = 16 * 1024;
const EVENT_CAPACITY: usize = 64;

/// TCP factory with an explicit destination and no fallback to production.
#[derive(Clone)]
pub struct TokioTcpTransportFactory {
    host: String,
    port: u16,
}

impl TokioTcpTransportFactory {
    pub fn new(host: impl Into<String>, port: u16) -> anyhow::Result<Self> {
        let host = host.into();
        if host.is_empty()
            || host.chars().any(|character| {
                character.is_whitespace()
                    || character.is_control()
                    || matches!(character, '/' | '\\' | '@' | '?' | '#')
            })
            || port == 0
        {
            bail!("TCP destination requires a host without URL syntax and a nonzero port");
        }
        Ok(Self { host, port })
    }
}

#[async_trait]
impl TransportFactory for TokioTcpTransportFactory {
    async fn create_transport(
        &self,
    ) -> anyhow::Result<(Arc<dyn Transport>, async_channel::Receiver<TransportEvent>)> {
        let stream = tokio::net::TcpStream::connect((self.host.as_str(), self.port))
            .await
            .context("TCP connection failed")?;
        stream.set_nodelay(true).context("setting TCP_NODELAY")?;
        Ok(from_byte_stream(stream))
    }
}

struct ByteStreamTransport<W> {
    writer: Mutex<Option<W>>,
    shutdown: watch::Sender<bool>,
}

impl<W> Drop for ByteStreamTransport<W> {
    fn drop(&mut self) {
        self.shutdown.send_replace(true);
    }
}

#[async_trait]
impl<W> Transport for ByteStreamTransport<W>
where
    W: AsyncWrite + Send + Unpin + 'static,
{
    async fn send(&self, data: Vec<u8>) -> anyhow::Result<()> {
        let mut writer = self.writer.lock().await;
        let Some(writer) = writer.as_mut() else {
            bail!("transport is closed");
        };
        writer.write_all(&data).await.context("TCP write failed")?;
        writer.flush().await.context("TCP flush failed")?;
        Ok(())
    }

    async fn disconnect(&self) {
        self.shutdown.send_replace(true);
        self.writer.lock().await.take();
    }
}

/// Build a [`Transport`] over any byte stream.
pub fn from_byte_stream<S>(
    stream: S,
) -> (Arc<dyn Transport>, async_channel::Receiver<TransportEvent>)
where
    S: AsyncRead + AsyncWrite + Send + Unpin + 'static,
{
    let (mut reader, writer) = tokio::io::split(stream);
    let (sender, receiver) = async_channel::bounded(EVENT_CAPACITY);
    let (shutdown, mut shutdown_rx) = watch::channel(false);
    let transport = Arc::new(ByteStreamTransport {
        writer: Mutex::new(Some(writer)),
        shutdown,
    });

    tokio::spawn(async move {
        let _ = sender.send(TransportEvent::Connected).await;
        let mut buffer = BytesMut::with_capacity(READ_CAPACITY);
        loop {
            tokio::select! {
                changed = shutdown_rx.changed() => {
                    if changed.is_err() || *shutdown_rx.borrow() {
                        break;
                    }
                }
                read = reader.read_buf(&mut buffer) => match read {
                    Ok(0) => break,
                    Ok(_) => {
                        let chunk = buffer.split().freeze();
                        if sender.send(TransportEvent::DataReceived(chunk)).await.is_err() {
                            break;
                        }
                    }
                    Err(_) => break,
                },
            }
        }
        let _ = sender.send(TransportEvent::Disconnected).await;
    });

    (transport, receiver)
}
