// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// PoC harness: muat QML, sediakan model 10k pesan ke konteks (persis pola
// QAbstractListModel→QML yang dipakai NeoChat/Tok), jalankan event loop.
#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QUrl>
#include "MessageModel.h"

int main(int argc, char *argv[]) {
    QGuiApplication app(argc, argv);
    QQmlApplicationEngine engine;
    MessageModel model;
    engine.rootContext()->setContextProperty("messageModel", &model);
    engine.load(QUrl::fromLocalFile(QStringLiteral(SRCDIR "/main.qml")));
    if (engine.rootObjects().isEmpty())
        return 1;
    return app.exec();
}
