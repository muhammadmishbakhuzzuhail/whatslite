# SPDX-License-Identifier: GPL-3.0-or-later
# PoC build — validasi QML ListView + C++ QAbstractListModel (10k var-height).
cmake_minimum_required(VERSION 3.16)
project(whatslite_qml_poc LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)

find_package(Qt6 REQUIRED COMPONENTS Core Network Gui Qml Quick QuickControls2)

add_executable(poc main.cpp MessageModel.h)
target_compile_definitions(poc PRIVATE SRCDIR="${CMAKE_CURRENT_SOURCE_DIR}")
target_link_libraries(poc PRIVATE Qt6::Gui Qt6::Qml Qt6::Quick Qt6::QuickControls2)

# Fase-3: klien bridge C++ (tanpa GUI) → uji call method + event vs mockengine Go.
add_executable(bridgepoc main_bridge.cpp WaEngineClient.cpp WaEngineClient.h)
target_link_libraries(bridgepoc PRIVATE Qt6::Core Qt6::Network)
