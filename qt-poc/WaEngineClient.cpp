// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
#include "WaEngineClient.h"
#include <QJsonDocument>

WaEngineClient::WaEngineClient(QObject *parent) : QObject(parent) {
    connect(&m_sock, &QLocalSocket::connected, this, &WaEngineClient::connected);
    connect(&m_sock, &QLocalSocket::readyRead, this, &WaEngineClient::onReadyRead);
    connect(&m_sock, &QLocalSocket::disconnected, this, [this] {
        // Putus → tolak semua request tertunda agar UI tak menggantung.
        const auto pending = m_pending;
        m_pending.clear();
        for (const auto &cb : pending)
            if (cb)
                cb(QJsonValue(), QStringLiteral("disconnected"));
        emit disconnected();
    });
}

void WaEngineClient::connectTo(const QString &sockPath) {
    m_sock.connectToServer(sockPath);
}

void WaEngineClient::call(const QString &method, const QJsonArray &args, Callback cb) {
    const QString id = QString::number(++m_seq);
    m_pending.insert(id, cb);
    writeFrame(QJsonObject{{"@type", method}, {"@extra", id}, {"args", args}});
}

void WaEngineClient::writeFrame(const QJsonObject &o) {
    QByteArray line = QJsonDocument(o).toJson(QJsonDocument::Compact);
    line.append('\n');
    m_sock.write(line);
    m_sock.flush();
}

void WaEngineClient::onReadyRead() {
    m_buf.append(m_sock.readAll());
    int nl;
    while ((nl = m_buf.indexOf('\n')) >= 0) {
        const QByteArray line = m_buf.left(nl);
        m_buf.remove(0, nl + 1);
        if (line.trimmed().isEmpty())
            continue;
        const QJsonObject o = QJsonDocument::fromJson(line).object();
        const QString type = o.value(QStringLiteral("@type")).toString();
        const QString extra = o.value(QStringLiteral("@extra")).toString();

        // Punya @extra + cocok pending → Response/Error sebuah request.
        if (!extra.isEmpty() && m_pending.contains(extra)) {
            Callback cb = m_pending.take(extra);
            if (type == QLatin1String("Error")) {
                QString msg = o.value(QStringLiteral("message")).toString();
                if (msg.isEmpty())
                    msg = o.value(QStringLiteral("code")).toString();
                if (cb)
                    cb(QJsonValue(), msg);
            } else if (cb) {
                cb(o.value(QStringLiteral("result")), QString());
            }
            continue;
        }
        // Selain itu: event push wa:*.
        if (type.startsWith(QLatin1String("wa:")))
            emit event(type, o.value(QStringLiteral("payload")));
    }
}
