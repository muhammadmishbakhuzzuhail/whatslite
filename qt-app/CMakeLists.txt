# SPDX-License-Identifier: GPL-3.0-or-later
# App QML WhatsLite (scaffold Fase-4). Reuse WaEngineClient dari qt-poc.
cmake_minimum_required(VERSION 3.16)
project(walite_qt LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)

find_package(Qt6 REQUIRED COMPONENTS Core Network Gui Qml Quick QuickControls2)

add_executable(walite-qt
    main.cpp
    AppController.h
    JsonListModel.h
    WaEngineClient.cpp
    WaEngineClient.h)
target_compile_definitions(walite-qt PRIVATE SRCDIR="${CMAKE_CURRENT_SOURCE_DIR}")
target_link_libraries(walite-qt PRIVATE
    Qt6::Core Qt6::Network Qt6::Gui Qt6::Qml Qt6::Quick Qt6::QuickControls2)
