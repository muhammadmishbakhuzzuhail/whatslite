// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// AppController — perekat QML↔engine. Memanggil method engine via WaEngineClient
// (GetChats/GetMessages), mengisi model, dan menerima event wa:* utk refresh.
// Q_INVOKABLE → dipanggil langsung dari QML (klik chat dst).
#pragma once
#include <QFile>
#include <QFileInfo>
#include <QJsonArray>
#include <QMimeDatabase>
#include <QObject>
#include <QUrl>
#include <QVariantList>
#include <QVariantMap>
#include "JsonListModel.h"
#include "WaEngineClient.h"

class AppController : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString mediaBase READ mediaBase NOTIFY mediaBaseChanged)
    Q_PROPERTY(bool keepDeleted READ keepDeleted NOTIFY keepDeletedChanged)
    Q_PROPERTY(QString state READ state NOTIFY stateChanged) // offline|connecting|ready
    Q_PROPERTY(QString qr READ qr NOTIFY qrChanged)          // data-URI QR (belum login)
    Q_PROPERTY(QVariantMap detail READ detail NOTIFY detailChanged) // objek detail (grup/profil)
    Q_PROPERTY(QString lastResult READ lastResult NOTIFY lastResultChanged) // hasil getter-string
public:
    AppController(WaEngineClient *c, JsonListModel *chats, JsonListModel *msgs,
                  JsonListModel *stickers, JsonListModel *gifs, JsonListModel *calls,
                  JsonListModel *starred, QObject *parent = nullptr)
        : QObject(parent), m_c(c), m_chats(chats), m_msgs(msgs), m_stickers(stickers),
          m_gifs(gifs), m_calls(calls), m_starred(starred) {
        connect(c, &WaEngineClient::connected, this, [this] {
            refreshChats();
            loadStickers();
            loadGifs();
            m_c->call(QStringLiteral("MediaBaseURL"), {}, [this](const QJsonValue &r, const QString &e) {
                if (e.isEmpty()) {
                    m_mediaBase = r.toString();
                    emit mediaBaseChanged();
                }
            });
            // Setelan anti-delete (fitur #2) → tampil di pane Setelan.
            m_c->call(QStringLiteral("GetKeepDeleted"), {}, [this](const QJsonValue &r, const QString &e) {
                if (e.isEmpty()) {
                    m_keepDeleted = r.toBool();
                    emit keepDeletedChanged();
                }
            });
            m_c->call(QStringLiteral("GetState"), {}, [this](const QJsonValue &r, const QString &e) {
                if (e.isEmpty()) { m_state = r.toString(); emit stateChanged(); }
            });
        });
        connect(c, &WaEngineClient::event, this, [this](const QString &t, const QJsonValue &p) {
            if (t == QLatin1String("wa:message")) {
                refreshChats(); // event = sinyal; FE re-fetch (pola engine)
                const QString jid = p.toString();
                if (!m_cur.isEmpty() && (jid.isEmpty() || jid == m_cur))
                    reloadMessages();
            } else if (t == QLatin1String("wa:qr")) {
                m_qr = p.toString(); emit qrChanged();
            } else if (t == QLatin1String("wa:state")) {
                m_state = p.toString(); emit stateChanged();
            } else if (t == QLatin1String("wa:ready")) {
                m_state = QStringLiteral("ready"); m_qr.clear();
                emit stateChanged(); emit qrChanged();
                refreshChats();
            } else if (t == QLatin1String("wa:loggedout")) {
                m_state = QStringLiteral("offline"); emit stateChanged();
            }
        });
    }

    Q_INVOKABLE void refreshChats() {
        m_c->call(QStringLiteral("GetChats"), {}, [this](const QJsonValue &r, const QString &e) {
            if (!e.isEmpty())
                return;
            m_chats->setItems(r.toArray());
            // Auto-buka chat pertama sekali (biar timeline langsung terisi).
            if (m_openFirst && m_chats->rowCount() > 0) {
                m_openFirst = false;
                openChat(m_chats->itemAt(0).value(QStringLiteral("id")).toString());
            }
        });
    }

    Q_INVOKABLE void openChat(const QString &id) {
        if (id.isEmpty())
            return;
        m_cur = id;
        reloadMessages();
    }

    // sendText mengirim teks ke chat yang sedang dibuka. Optimistik: setelah
    // engine balas, muat ulang timeline (engine juga emit wa:message → reload).
    Q_INVOKABLE void sendText(const QString &text) {
        if (m_cur.isEmpty() || text.trimmed().isEmpty())
            return;
        m_c->call(QStringLiteral("SendText"), QJsonArray{m_cur, text},
                  [this](const QJsonValue &, const QString &e) {
                      if (e.isEmpty())
                          reloadMessages();
                  });
    }

    // loadStickers menarik koleksi stiker tersimpan (fitur CRUD #1) ke model picker.
    Q_INVOKABLE void loadStickers() {
        m_c->call(QStringLiteral("ListSavedStickers"), {}, [this](const QJsonValue &r, const QString &e) {
            if (e.isEmpty())
                m_stickers->setItems(r.toArray());
        });
    }

    // sendSticker mengirim stiker koleksi (by hash) ke chat aktif → reload timeline.
    Q_INVOKABLE void sendSticker(const QString &hash) {
        if (m_cur.isEmpty() || hash.isEmpty())
            return;
        m_c->call(QStringLiteral("SendSavedSticker"), QJsonArray{m_cur, hash},
                  [this](const QJsonValue &, const QString &e) {
                      if (e.isEmpty())
                          reloadMessages();
                  });
    }

    // --- GIF (fitur #3, pola identik stiker) ---
    Q_INVOKABLE void loadGifs() {
        m_c->call(QStringLiteral("ListSavedGifs"), {}, [this](const QJsonValue &r, const QString &e) {
            if (e.isEmpty())
                m_gifs->setItems(r.toArray());
        });
    }
    Q_INVOKABLE void sendGif(const QString &hash) {
        if (m_cur.isEmpty() || hash.isEmpty())
            return;
        m_c->call(QStringLiteral("SendSavedGif"), QJsonArray{m_cur, hash},
                  [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) reloadMessages(); });
    }

    // --- Aksi pesan (context menu) ---
    Q_INVOKABLE void react(const QString &msgId, const QString &sender, bool fromMe, const QString &emoji) {
        m_c->call(QStringLiteral("React"), QJsonArray{m_cur, msgId, sender, emoji, fromMe},
                  [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) reloadMessages(); });
    }
    Q_INVOKABLE void star(const QString &msgId, const QString &sender, bool fromMe, bool on) {
        m_c->call(QStringLiteral("StarMessage"), QJsonArray{m_cur, msgId, sender, fromMe, on}, {});
    }
    Q_INVOKABLE void deleteMsg(const QString &msgId, const QString &sender, bool fromMe, bool everyone) {
        m_c->call(QStringLiteral("DeleteMessage"), QJsonArray{m_cur, msgId, sender, fromMe, everyone},
                  [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) reloadMessages(); });
    }
    // Simpan stiker/GIF yang dikirim teman ke koleksi (fitur #1/#3, dari bubble).
    Q_INVOKABLE void saveStickerFromMsg(const QString &msgId) {
        m_c->call(QStringLiteral("SaveSticker"), QJsonArray{m_cur, msgId},
                  [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) loadStickers(); });
    }
    Q_INVOKABLE void saveGifFromMsg(const QString &msgId) {
        m_c->call(QStringLiteral("SaveGif"), QJsonArray{m_cur, msgId},
                  [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) loadGifs(); });
    }

    // --- Pane lain (riwayat panggilan, pesan berbintang) ---
    Q_INVOKABLE void loadCalls() {
        m_c->call(QStringLiteral("GetCalls"), {}, [this](const QJsonValue &r, const QString &e) {
            if (e.isEmpty()) m_calls->setItems(r.toArray());
        });
    }
    Q_INVOKABLE void loadStarred() {
        m_c->call(QStringLiteral("GetStarred"), {}, [this](const QJsonValue &r, const QString &e) {
            if (e.isEmpty()) m_starred->setItems(r.toArray());
        });
    }

    // --- Loader GENERIK: pane read-only apa pun (GetX 0-arg) → JsonListModel.
    // QML: app.loadInto("GetStatuses", statusModel). Skala tanpa method baru. ---
    Q_INVOKABLE void loadInto(const QString &method, QObject *model) {
        auto *m = qobject_cast<JsonListModel *>(model);
        if (!m)
            return;
        m_c->call(method, {}, [m](const QJsonValue &r, const QString &e) {
            if (e.isEmpty()) m->setItems(r.toArray());
        });
    }

    // --- Pencarian pesan (FTS) ---
    Q_INVOKABLE void search(const QString &q, QObject *model) {
        auto *m = qobject_cast<JsonListModel *>(model);
        if (!m)
            return;
        if (q.trimmed().isEmpty()) {
            m->setItems({});
            return;
        }
        m_c->call(QStringLiteral("SearchMessages"), QJsonArray{q}, [m](const QJsonValue &r, const QString &e) {
            if (e.isEmpty()) m->setItems(r.toArray());
        });
    }

    // --- Detail objek tunggal (grup-info, profil kontak) → property `detail` ---
    Q_INVOKABLE void loadDetail(const QString &method, const QString &arg) {
        QJsonArray a;
        if (!arg.isEmpty())
            a.append(arg);
        m_c->call(method, a, [this](const QJsonValue &r, const QString &e) {
            if (e.isEmpty()) {
                m_detail = r.toObject().toVariantMap();
                emit detailChanged();
            }
        });
    }

    // Varian multi-arg (mis. GetMessageInfo(chat,msgId), GetPrivacy()).
    Q_INVOKABLE void loadDetailA(const QString &method, const QVariantList &args) {
        m_c->call(method, QJsonArray::fromVariantList(args), [this](const QJsonValue &r, const QString &e) {
            if (e.isEmpty()) {
                m_detail = r.toObject().toVariantMap();
                emit detailChanged();
            }
        });
    }

    // --- Privasi ---
    Q_INVOKABLE void setPrivacy(const QString &name, const QString &value) {
        m_c->call(QStringLiteral("SetPrivacy"), QJsonArray{name, value}, {});
    }

    // --- Kirim dokumen (file → base64 data-URI → SendMedia), dgn nama (rename) ---
    Q_INVOKABLE void sendDocument(const QUrl &fileUrl, const QString &fileName) {
        if (m_cur.isEmpty())
            return;
        const QString path = fileUrl.toLocalFile();
        QFile f(path);
        if (!f.open(QIODevice::ReadOnly))
            return;
        const QByteArray bytes = f.readAll();
        f.close();
        const QString mime = QMimeDatabase().mimeTypeForFile(path).name();
        const QString dataUri = QStringLiteral("data:") + mime + QStringLiteral(";base64,") + QString::fromLatin1(bytes.toBase64());
        const QString name = fileName.isEmpty() ? QFileInfo(path).fileName() : fileName;
        // SendMedia(jid, kind, caption, fileName, dataURI, viewOnce, seconds)
        m_c->call(QStringLiteral("SendMedia"), QJsonArray{m_cur, "document", "", name, dataUri, false, 0},
                  [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) reloadMessages(); });
    }

    // --- Teruskan pesan (context-menu → pilih chat) ---
    Q_INVOKABLE void forwardMsg(const QString &msgId, const QString &toJid) {
        if (m_cur.isEmpty() || msgId.isEmpty() || toJid.isEmpty())
            return;
        m_c->call(QStringLiteral("Forward"), QJsonArray{m_cur, msgId, toJid}, {});
    }

    // --- Koneksi / login / logout ---
    Q_INVOKABLE void doConnect() { m_c->call(QStringLiteral("Connect"), {}, {}); }
    Q_INVOKABLE void refreshQr() { m_c->call(QStringLiteral("MyQR"), {}, [this](const QJsonValue &r, const QString &e) {
        if (e.isEmpty()) { m_qr = r.toString(); emit qrChanged(); } }); }
    Q_INVOKABLE void logout() { m_c->call(QStringLiteral("Logout"), {}, {}); }

    // --- Kelola chat (target = chat aktif kecuali diberi jid) ---
    Q_INVOKABLE void markRead(const QString &jid) { m_c->call(QStringLiteral("MarkRead"), QJsonArray{jid}, [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) refreshChats(); }); }
    Q_INVOKABLE void pinChat(const QString &jid, bool on) { m_c->call(QStringLiteral("Pin"), QJsonArray{jid, on}, [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) refreshChats(); }); }
    Q_INVOKABLE void muteChat(const QString &jid, bool on) { m_c->call(QStringLiteral("Mute"), QJsonArray{jid, on}, [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) refreshChats(); }); }
    Q_INVOKABLE void archiveChat(const QString &jid, bool on) { m_c->call(QStringLiteral("Archive"), QJsonArray{jid, on}, [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) refreshChats(); }); }
    Q_INVOKABLE void deleteChat(const QString &jid) { m_c->call(QStringLiteral("DeleteChat"), QJsonArray{jid}, [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) refreshChats(); }); }

    // --- Aksi pesan tambahan ---
    Q_INVOKABLE void replyText(const QString &quotedId, const QString &quotedSender, const QString &quotedText, const QString &text) {
        if (m_cur.isEmpty() || text.trimmed().isEmpty()) return;
        m_c->call(QStringLiteral("Reply"), QJsonArray{m_cur, text, quotedId, quotedSender, quotedText}, [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) reloadMessages(); });
    }
    Q_INVOKABLE void editMessage(const QString &msgId, const QString &newText) {
        m_c->call(QStringLiteral("EditMessage"), QJsonArray{m_cur, msgId, newText}, [this](const QJsonValue &, const QString &e) { if (e.isEmpty()) reloadMessages(); });
    }
    Q_INVOKABLE void pinMessage(const QString &msgId, const QString &sender, bool fromMe, bool on) {
        m_c->call(QStringLiteral("PinMessage"), QJsonArray{m_cur, msgId, sender, fromMe, on}, {});
    }
    Q_INVOKABLE void sendTyping(bool on) {
        if (!m_cur.isEmpty()) m_c->call(QStringLiteral("SendTyping"), QJsonArray{m_cur, on}, {});
    }

    // --- Pagination: muat pesan lebih lama (prepend) ---
    Q_INVOKABLE void loadOlder() {
        if (m_cur.isEmpty() || m_oldestTs <= 0) return;
        m_c->call(QStringLiteral("GetMessagesBefore"), QJsonArray{m_cur, m_oldestTs}, [this](const QJsonValue &r, const QString &e) {
            if (e.isEmpty()) {
                const QJsonArray older = r.toArray();
                if (!older.isEmpty()) {
                    m_oldestTs = older.first().toObject().value(QStringLiteral("ts")).toVariant().toLongLong();
                    m_msgs->prepend(older);
                }
            }
        });
    }

    // --- Setelan anti-delete (fitur #2) ---
    Q_INVOKABLE void setKeepDeleted(bool v) {
        m_c->call(QStringLiteral("SetKeepDeleted"), QJsonArray{v}, {});
        m_keepDeleted = v;
        emit keepDeletedChanged();
    }

    // --- Helper generik: jangkau method engine apa pun dari QML ---
    // act: aksi fire-and-forget. actReload: aksi lalu muat ulang chat/pesan.
    // loadIntoA: getter-list berargumen → model. fetchStr: getter string → lastResult.
    Q_INVOKABLE void act(const QString &method, const QVariantList &args) {
        m_c->call(method, QJsonArray::fromVariantList(args), {});
    }
    Q_INVOKABLE void actReload(const QString &method, const QVariantList &args) {
        m_c->call(method, QJsonArray::fromVariantList(args), [this](const QJsonValue &, const QString &e) {
            if (e.isEmpty()) { reloadMessages(); refreshChats(); }
        });
    }
    Q_INVOKABLE void loadIntoA(const QString &method, const QVariantList &args, QObject *model) {
        auto *m = qobject_cast<JsonListModel *>(model);
        if (!m) return;
        m_c->call(method, QJsonArray::fromVariantList(args), [m](const QJsonValue &r, const QString &e) {
            if (e.isEmpty()) m->setItems(r.toArray());
        });
    }
    Q_INVOKABLE void fetchStr(const QString &method, const QVariantList &args) {
        m_c->call(method, QJsonArray::fromVariantList(args), [this](const QJsonValue &r, const QString &e) {
            if (e.isEmpty()) { m_lastResult = r.toString(); emit lastResultChanged(); }
        });
    }

    QString mediaBase() const { return m_mediaBase; }
    bool keepDeleted() const { return m_keepDeleted; }
    QString lastResult() const { return m_lastResult; }
    QString state() const { return m_state; }
    QString qr() const { return m_qr; }
    QVariantMap detail() const { return m_detail; }

signals:
    void mediaBaseChanged();
    void keepDeletedChanged();
    void stateChanged();
    void qrChanged();
    void detailChanged();
    void lastResultChanged();

private:
    void reloadMessages() {
        if (m_cur.isEmpty())
            return;
        m_c->call(QStringLiteral("GetMessages"), QJsonArray{m_cur}, [this](const QJsonValue &r, const QString &e) {
            if (e.isEmpty()) {
                const QJsonArray msgs = r.toArray();
                m_msgs->setItems(msgs);
                m_oldestTs = msgs.isEmpty() ? 0 : msgs.first().toObject().value(QStringLiteral("ts")).toVariant().toLongLong();
            }
        });
    }

    WaEngineClient *m_c;
    JsonListModel *m_chats;
    JsonListModel *m_msgs;
    JsonListModel *m_stickers;
    JsonListModel *m_gifs;
    JsonListModel *m_calls;
    JsonListModel *m_starred;
    QString m_cur;
    QString m_mediaBase;
    QString m_state;
    QString m_qr;
    QVariantMap m_detail;
    QString m_lastResult;
    qint64 m_oldestTs = 0;
    bool m_keepDeleted = true;
    bool m_openFirst = true;
};
