// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// JsonListModel — model generik: tampung QJsonArray (hasil DTO dari engine) dan
// ekspor tiap objek ke QML sebagai role "m" (QVariantMap). Satu kelas dipakai
// ulang utk daftar chat MAUPUN pesan → tak perlu C++ per-DTO; delegate akses
// langsung m.name / m.text / m.dir dst. Cocok dgn kontrak engine yang JSON-shaped.
#pragma once
#include <QAbstractListModel>
#include <QJsonArray>
#include <QJsonObject>

class JsonListModel : public QAbstractListModel {
    Q_OBJECT
public:
    enum Roles { ItemRole = Qt::UserRole + 1 };
    explicit JsonListModel(QObject *parent = nullptr) : QAbstractListModel(parent) {}

    void setItems(const QJsonArray &a) {
        beginResetModel();
        m_items = a;
        endResetModel();
    }
    QJsonObject itemAt(int i) const {
        return (i >= 0 && i < m_items.size()) ? m_items.at(i).toObject() : QJsonObject();
    }

    int rowCount(const QModelIndex & = QModelIndex()) const override { return m_items.size(); }
    QVariant data(const QModelIndex &idx, int role) const override {
        if (!idx.isValid() || idx.row() >= m_items.size())
            return {};
        if (role == ItemRole)
            return m_items.at(idx.row()).toObject().toVariantMap();
        return {};
    }
    QHash<int, QByteArray> roleNames() const override { return {{ItemRole, "m"}}; }

private:
    QJsonArray m_items;
};
