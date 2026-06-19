// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// Harness uji Fase-3 (tanpa GUI, QCoreApplication): connect ke bridge, call
// GetChats, dengar satu event wa:* → cetak BRIDGE_OK. Membuktikan klien Qt
// bicara protokol bridge engine yang sebenarnya.
#include "WaEngineClient.h"
#include <QCoreApplication>
#include <QJsonArray>
#include <QJsonDocument>
#include <QTimer>
#include <QtGlobal>

int main(int argc, char *argv[]) {
    QCoreApplication app(argc, argv);
    const QString sock = argc > 1 ? QString::fromLocal8Bit(argv[1]) : QStringLiteral("/tmp/bridge.sock");

    WaEngineClient client;
    bool gotChats = false, gotEvent = false;
    auto maybeQuit = [&] {
        if (gotChats && gotEvent) {
            qInfo("BRIDGE_OK");
            app.quit();
        }
    };

    QObject::connect(&client, &WaEngineClient::connected, [&] {
        client.call(QStringLiteral("GetChats"), {}, [&](const QJsonValue &res, const QString &err) {
            if (!err.isEmpty()) {
                qWarning("CALL_ERR %s", qUtf8Printable(err));
                app.exit(1);
                return;
            }
            qInfo("CHATS %s", QJsonDocument(res.toArray()).toJson(QJsonDocument::Compact).constData());
            gotChats = true;
            maybeQuit();
        });
    });
    QObject::connect(&client, &WaEngineClient::event, [&](const QString &t, const QJsonValue &p) {
        if (!gotEvent) {
            qInfo("EVENT %s %s", qUtf8Printable(t), qUtf8Printable(p.toString()));
            gotEvent = true;
            maybeQuit();
        }
    });

    client.connectTo(sock);
    QTimer::singleShot(5000, [&] { qWarning("TIMEOUT"); app.exit(2); });
    return app.exec();
}
