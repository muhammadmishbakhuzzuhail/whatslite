// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// WaEngineClient — klien C++/Qt untuk jembatan engine (NDJSON/UDS). Dipakai
// ulang di app Qt nyata: connect ke bridge.sock, call method engine (korelasi
// @extra → callback), dengar event wa:* sebagai signal. Mirror model TDLib
// (send any-thread / satu stream response+event).
#pragma once
#include <QByteArray>
#include <QHash>
#include <QJsonArray>
#include <QJsonObject>
#include <QJsonValue>
#include <QLocalSocket>
#include <QObject>
#include <QString>
#include <functional>

class WaEngineClient : public QObject {
    Q_OBJECT
public:
    using Callback = std::function<void(const QJsonValue &result, const QString &error)>;

    explicit WaEngineClient(QObject *parent = nullptr);
    void connectTo(const QString &sockPath);

    // call mengirim request; cb dipanggil saat Response/Error tiba (via @extra).
    void call(const QString &method, const QJsonArray &args, Callback cb);

signals:
    void connected();
    void disconnected();
    void event(const QString &type, const QJsonValue &payload); // wa:*

private slots:
    void onReadyRead();

private:
    void writeFrame(const QJsonObject &o);

    // Urutan deklarasi PENTING: m_sock TERAKHIR → dihancurkan PERTAMA (urutan
    // destruksi terbalik). ~QLocalSocket emit disconnected() → handler akses
    // m_pending; jadi m_pending HARUS masih hidup (dideklarasi sebelum m_sock).
    QByteArray m_buf;
    quint64 m_seq = 0;
    QHash<QString, Callback> m_pending;
    QLocalSocket m_sock;
};
