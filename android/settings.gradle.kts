pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

plugins {
    // Auto-provision the Java 17 toolchain (downloads into ~/.gradle/jdks)
    // so no system-wide JDK 17 install is needed.
    id("org.gradle.toolchains.foojay-resolver-convention") version "0.9.0"
}

dependencyResolutionManagement {
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "TuiGo"

include(":engine")
include(":app")
