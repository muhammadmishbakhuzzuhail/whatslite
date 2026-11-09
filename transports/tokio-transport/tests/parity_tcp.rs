#![cfg(feature = "tcp")]
//! Phase 5 raw TCP transport contract.

use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpListener;
use wa_rs_tokio_transport::{TokioTcpTransportFactory, from_byte_stream};
use wacore::net::{Transport, TransportEvent, TransportFactory};

#[tokio::test]
async fn tcp_factory_roundtrips_bytes() {
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    let server = tokio::spawn(async move {
        let (mut socket, _) = listener.accept().await.unwrap();
        let mut request = [0u8; 4];
        socket.read_exact(&mut request).await.unwrap();
        assert_eq!(&request, b"ping");
        socket.write_all(b"pong").await.unwrap();
        socket.flush().await.unwrap();
        let mut end = [0u8; 1];
        let _ = socket.read(&mut end).await;
    });

    let factory = TokioTcpTransportFactory::new("127.0.0.1", address.port()).unwrap();
    let (transport, events) = factory.create_transport().await.unwrap();
    assert!(matches!(
        events.recv().await.unwrap(),
        TransportEvent::Connected
    ));

    transport.send(b"ping".to_vec()).await.unwrap();
    match events.recv().await.unwrap() {
        TransportEvent::DataReceived(bytes) => assert_eq!(&bytes[..], b"pong"),
        other => panic!("unexpected event: {other:?}"),
    }
    transport.disconnect().await;
    server.await.unwrap();
}

#[tokio::test]
async fn from_byte_stream_roundtrips_duplex() {
    let (client, mut server) = tokio::io::duplex(64);
    let (transport, events) = from_byte_stream(client);
    assert!(matches!(
        events.recv().await.unwrap(),
        TransportEvent::Connected
    ));

    transport.send(b"abc".to_vec()).await.unwrap();
    let mut request = [0u8; 3];
    server.read_exact(&mut request).await.unwrap();
    assert_eq!(&request, b"abc");

    server.write_all(b"xyz").await.unwrap();
    match events.recv().await.unwrap() {
        TransportEvent::DataReceived(bytes) => assert_eq!(&bytes[..], b"xyz"),
        other => panic!("unexpected event: {other:?}"),
    }
    transport.disconnect().await;
}

#[test]
fn tcp_factory_rejects_invalid_destinations() {
    assert!(TokioTcpTransportFactory::new("", 443).is_err());
    assert!(TokioTcpTransportFactory::new("https://example.test", 443).is_err());
    assert!(TokioTcpTransportFactory::new("example.test", 0).is_err());
    assert!(TokioTcpTransportFactory::new("example.test", 443).is_ok());
}
