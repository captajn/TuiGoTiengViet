# fcitx5 addon for Tui Go (Sen mode, no preedit).
#
# Layout:
#   go/   — C bridge to Tui Go's Go typing engine, built as a C archive.
#   src/  — Sen state machine (no fcitx5) + fcitx5 shell.
#   test/ — Sen machine + real Go engine + fake sink; runs without fcitx5.
#
# The addon target is only built when fcitx5 dev files are present. The
# Sen machine and its test always build, so the typing logic is verifiable
# on machines without fcitx5.
cmake_minimum_required(VERSION 3.10)
project(tuigo-fcitx5)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_program(GO_EXECUTABLE go REQUIRED)

# --- Go typing engine as a C archive -------------------------------------
set(TUIGO_SHIM_ARCHIVE "${CMAKE_BINARY_DIR}/tuigo-shim.a")
set(TUIGO_SHIM_HEADER "${CMAKE_BINARY_DIR}/tuigo-shim.h")
add_custom_command(
    OUTPUT "${TUIGO_SHIM_ARCHIVE}" "${TUIGO_SHIM_HEADER}"
    DEPENDS "${CMAKE_CURRENT_SOURCE_DIR}/go/shim.go"
            "${CMAKE_CURRENT_SOURCE_DIR}/go/go.mod"
    WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}/go"
    COMMAND ${CMAKE_COMMAND} -E env CGO_CFLAGS=-fPIC CGO_LDFLAGS=-fPIC
            "${GO_EXECUTABLE}" build -buildmode=c-archive -ldflags=-s\ -w
            -o "${TUIGO_SHIM_ARCHIVE}" .
    VERBATIM)
add_custom_target(tuigo_shim_archive DEPENDS "${TUIGO_SHIM_ARCHIVE}" "${TUIGO_SHIM_HEADER}")

add_library(TuiGoShim STATIC IMPORTED GLOBAL)
add_dependencies(TuiGoShim tuigo_shim_archive)
file(MAKE_DIRECTORY "${CMAKE_BINARY_DIR}")
set_target_properties(TuiGoShim PROPERTIES
    IMPORTED_LOCATION "${TUIGO_SHIM_ARCHIVE}"
    INTERFACE_INCLUDE_DIRECTORIES "${CMAKE_BINARY_DIR}")

# --- Sen machine (no fcitx5) --------------------------------------------
add_library(tuigo_sen STATIC
    src/sen.cpp
    src/utf8.cpp
    src/quirks.cpp
    src/tuigo-engine.cpp)
# The addon is a shared module linking this static lib: its objects must be
# position-independent, or the final link fails with R_X86_64_PC32.
set_target_properties(tuigo_sen PROPERTIES POSITION_INDEPENDENT_CODE ON)
target_include_directories(tuigo_sen PUBLIC "${CMAKE_CURRENT_SOURCE_DIR}/src")
target_link_libraries(tuigo_sen PUBLIC TuiGoShim)
add_dependencies(tuigo_sen tuigo_shim_archive)

# --- Sen test: real Go engine + fake sink, no fcitx5 --------------------
add_executable(sen_test test/sen_test.cpp)
target_link_libraries(sen_test PRIVATE tuigo_sen)
enable_testing()
add_test(NAME sen_test COMMAND sen_test)

# --- fcitx5 addon ---------------------------------------------------------
find_package(Fcitx5Core QUIET)
find_package(Fcitx5Config QUIET)
if(Fcitx5Core_FOUND AND Fcitx5Config_FOUND)
    add_library(tuigo MODULE
        src/addon.cpp
        src/addon-factory.cpp)
    set_target_properties(tuigo PROPERTIES POSITION_INDEPENDENT_CODE ON)
    target_link_libraries(tuigo PRIVATE tuigo_sen Fcitx5::Core Fcitx5::Config)

    include(GNUInstallDirs)
    install(TARGETS tuigo DESTINATION "${CMAKE_INSTALL_LIBDIR}/fcitx5")
    # NOTE: do NOT use fcitx5_translate_desktop_file here. It only registers
    # a custom command for the output, and nothing depends on it, so the
    # file is never generated during build and install fails with
    # "file INSTALL cannot find .../tuigo.conf". These conf files contain
    # no translatable strings, so a plain configure-time copy is exact.
    configure_file(tuigo.conf.in "${CMAKE_CURRENT_BINARY_DIR}/tuigo.conf" COPYONLY)
    configure_file(tuigo-addon.conf.in.in "${CMAKE_CURRENT_BINARY_DIR}/tuigo-addon.conf" COPYONLY)
    install(FILES "${CMAKE_CURRENT_BINARY_DIR}/tuigo.conf"
            DESTINATION "${CMAKE_INSTALL_DATADIR}/fcitx5/inputmethod")
    install(FILES "${CMAKE_CURRENT_BINARY_DIR}/tuigo-addon.conf"
            RENAME tuigo.conf DESTINATION "${FCITX_INSTALL_PKGDATADIR}/addon")
    # Tray icons: TuiGo's own E/V badge (app/tray_icon.go), ported to
    # PNG at the sizes the tray asks for. PNG, not SVG: an SVG icon
    # never showed up in the tray (fcitx5's tray renderer needs an SVG
    # loader the system may not have); PNG loads natively. Install each
    # size to BOTH apps/ and status/: the tray/status area looks icons
    # up in the status context (Ngó Sen does the same).
    foreach(_size 22 24 48 256)
        foreach(_mode vi en)
            set(_icon "${CMAKE_CURRENT_SOURCE_DIR}/data/icons/fcitx-tuigo-${_mode}-${_size}.png")
            if(EXISTS "${_icon}")
            install(FILES "${_icon}"
                RENAME "fcitx-tuigo-${_mode}.png"
                DESTINATION "${CMAKE_INSTALL_DATADIR}/icons/hicolor/${_size}x${_size}/apps")
            install(FILES "${_icon}"
                RENAME "fcitx-tuigo-${_mode}.png"
                DESTINATION "${CMAKE_INSTALL_DATADIR}/icons/hicolor/${_size}x${_size}/status")
            endif()
        endforeach()
    endforeach()
else()
    message(STATUS "Fcitx5 dev files not found: skipping addon target (sen_test still builds)")
endif()

# --- Packaging (CPack: DEB + RPM) -----------------------------------------
# Version: -DTUIGO_VERSION=... (CI, from the release tag or appVersion)
# or parsed from ../../app/main.go (local builds).
if(NOT DEFINED TUIGO_VERSION OR TUIGO_VERSION STREQUAL "")
    set(TUIGO_VERSION "0.0.0")
    if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/../app/main.go")
        file(READ "${CMAKE_CURRENT_SOURCE_DIR}/../app/main.go" _tuigo_main_go)
        string(REGEX MATCH "appVersion = \"([0-9]+\\.[0-9]+\\.[0-9]+)\"" _tuigo_ver_match
               "${_tuigo_main_go}")
        if(CMAKE_MATCH_1)
            set(TUIGO_VERSION "${CMAKE_MATCH_1}")
        endif()
    endif()
endif()
set(CPACK_PACKAGE_NAME "fcitx5-tuigo")
set(CPACK_PACKAGE_VENDOR "TuiGo")
set(CPACK_PACKAGE_VERSION "${TUIGO_VERSION}")
# Split parts, explicitly: CPack otherwise derives them from the
# (unset) project version and reports 0/1/1 for a 0.0.9 package.
string(REGEX MATCH "^([0-9]+)\\.([0-9]+)\\.([0-9]+)" _tuigo_ver_parts "${TUIGO_VERSION}")
set(CPACK_PACKAGE_VERSION_MAJOR "${CMAKE_MATCH_1}")
set(CPACK_PACKAGE_VERSION_MINOR "${CMAKE_MATCH_2}")
set(CPACK_PACKAGE_VERSION_PATCH "${CMAKE_MATCH_3}")
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY
    "Tui Go Vietnamese input method for fcitx5 (Sen mode, no preedit)")
set(CPACK_PACKAGE_DESCRIPTION
    "Tui Go (TuiGoTiengViet) addon for fcitx5. Sen mode: text is committed immediately, no preedit; tone/mark edits rewrite committed text via deleteSurroundingText or forwarded BackSpace.")
set(CPACK_PACKAGE_LICENSE "GPL-3.0-or-later")
set(CPACK_PACKAGE_CONTACT "TuiGo contributors")
set(CPACK_PACKAGE_HOMEPAGE_URL "https://github.com/captajn/TuiGoTiengViet")
# The addon must live where fcitx5 looks for it (/usr, not /usr/local).
set(CPACK_PACKAGING_INSTALL_PREFIX "/usr")
set(CPACK_DEBIAN_PACKAGE_MAINTAINER "TuiGo contributors")
set(CPACK_DEBIAN_PACKAGE_DEPENDS "fcitx5 (>= 5.0)")
set(CPACK_DEBIAN_PACKAGE_SECTION "utils")
set(CPACK_DEBIAN_FILE_NAME "DEB-DEFAULT")
set(CPACK_RPM_PACKAGE_LICENSE "GPL-3.0-or-later")
set(CPACK_RPM_PACKAGE_GROUP "System/Utilities")
set(CPACK_RPM_PACKAGE_REQUIRES "fcitx5 >= 5.0")
set(CPACK_RPM_FILE_NAME "RPM-DEFAULT")
# No CPACK_GENERATOR here: the CI workflow passes -G DEB / -G RPM
# explicitly, and a plain local `cpack` falls back to the platform
# default (TGZ) instead of failing on a missing rpmbuild.
include(CPack)
