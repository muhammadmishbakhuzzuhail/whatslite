// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// PoC: model 10k pesan tinggi-variabel untuk menguji virtualisasi QML ListView.
// Bentuk meniru kontrak engine (text + arah in/out) supaya delegate-nya realistis.
#pragma once
#include <QAbstractListModel>
#include <QHash>
#include <QString>
#include <vector>

class MessageModel : public QAbstractListModel {
    Q_OBJECT
public:
    enum Roles { TextRole = Qt::UserRole + 1, OutRole };

    explicit MessageModel(QObject *parent = nullptr) : QAbstractListModel(parent) {
        m_rows.reserve(10000);
        for (int i = 0; i < 10000; ++i) {
            // Panjang teks bervariasi 1..40 "kata" → tinggi bubble berbeda-beda
            // (kasus terberat ListView: tinggi delegate tak seragam).
            QString t;
            const int words = 1 + (i * 7) % 40;
            for (int w = 0; w < words; ++w)
                t += QStringLiteral("kata%1 ").arg(w);
            m_rows.push_back({t.trimmed(), (i % 2) == 0});
        }
    }

    int rowCount(const QModelIndex & = QModelIndex()) const override {
        return static_cast<int>(m_rows.size());
    }
    QVariant data(const QModelIndex &idx, int role) const override {
        if (!idx.isValid() || idx.row() >= rowCount())
            return {};
        const Row &r = m_rows[idx.row()];
        switch (role) {
        case TextRole: return r.text;
        case OutRole:  return r.out;
        }
        return {};
    }
    QHash<int, QByteArray> roleNames() const override {
        return {{TextRole, "mtext"}, {OutRole, "mout"}};
    }

private:
    struct Row { QString text; bool out; };
    std::vector<Row> m_rows;
};
