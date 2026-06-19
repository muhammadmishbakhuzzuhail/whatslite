// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// Entry app QML WhatsLite: wire WaEngineClient → model chat/pesan → AppController
// → QML. Connect ke bridge.sock engine. WALITE_SELFTEST=1 → cetak jumlah lalu
// keluar (uji headless tanpa display).
#include "AppController.h"
#include "JsonListModel.h"
#include "WaEngineClient.h"
#include <QGuiApplication>
#include <QImage>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QQuickWindow>
#include <QTimer>
#include <QUrl>

int main(int argc, char *argv[]) {
    QGuiApplication app(argc, argv);
    const QString sock = argc > 1 ? QString::fromLocal8Bit(argv[1]) : QStringLiteral("/tmp/bridge.sock");

    WaEngineClient client;
    JsonListModel chatsModel, msgsModel, stickersModel, gifsModel, callsModel, starredModel;
    AppController ctrl(&client, &chatsModel, &msgsModel, &stickersModel, &gifsModel,
                       &callsModel, &starredModel);

    QQmlApplicationEngine engine;
    QQmlContext *ctx = engine.rootContext();
    ctx->setContextProperty("chatsModel", &chatsModel);
    ctx->setContextProperty("msgsModel", &msgsModel);
    ctx->setContextProperty("stickersModel", &stickersModel);
    ctx->setContextProperty("gifsModel", &gifsModel);
    ctx->setContextProperty("callsModel", &callsModel);
    ctx->setContextProperty("starredModel", &starredModel);
    // Pane generik (diisi via app.loadInto / app.search).
    static JsonListModel statusModel, contactsModel, searchModel;
    static JsonListModel channelsModel, communitiesModel, archivedModel, scheduledModel;
    ctx->setContextProperty("statusModel", &statusModel);
    ctx->setContextProperty("contactsModel", &contactsModel);
    ctx->setContextProperty("searchModel", &searchModel);
    ctx->setContextProperty("channelsModel", &channelsModel);
    ctx->setContextProperty("communitiesModel", &communitiesModel);
    ctx->setContextProperty("archivedModel", &archivedModel);
    ctx->setContextProperty("scheduledModel", &scheduledModel);
    ctx->setContextProperty("app", &ctrl);
    ctx->setContextProperty("startDark", qEnvironmentVariableIsSet("WALITE_DARK"));
    ctx->setContextProperty("openPanel", qEnvironmentVariable("WALITE_OPEN"));
    ctx->setContextProperty("startLock", qEnvironmentVariableIsSet("WALITE_LOCK"));

    engine.load(QUrl::fromLocalFile(QStringLiteral(SRCDIR "/main.qml")));
    if (engine.rootObjects().isEmpty())
        return 1;

    client.connectTo(sock);

    if (qEnvironmentVariableIsSet("WALITE_SELFTEST")) {
        // Tahap 1: kirim teks + stiker koleksi (uji jalur TULIS + fitur stiker).
        QTimer::singleShot(2500, [&] {
            ctrl.sendText(QStringLiteral("Halo dari Qt!"));
            ctrl.sendSticker(QStringLiteral("bbb"));
        });
        // Tahap 2: semua bolak-balik via bridge → hitung + screenshot.
        QTimer::singleShot(3800, [&] {
            qInfo("UI_OK chats=%d msgs=%d stickers=%d", chatsModel.rowCount(),
                  msgsModel.rowCount(), stickersModel.rowCount());
            if (qEnvironmentVariableIsSet("WALITE_SHOT") && !engine.rootObjects().isEmpty()) {
                if (auto *w = qobject_cast<QQuickWindow *>(engine.rootObjects().first())) {
                    const QImage img = w->grabWindow();
                    if (img.save(QStringLiteral("/tmp/walite.png")))
                        qInfo("SHOT %dx%d", img.width(), img.height());
                }
            }
            app.quit();
        });
    }
    return app.exec();
}
