import java.util.Properties

plugins {
    id("com.android.application")
    kotlin("android")
}

// Release signing — keystore.properties + keystore/ are gitignored; if absent,
// the release build falls back to unsigned.
val keystoreProps = Properties().apply {
    val f = rootProject.file("keystore.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}
val hasReleaseKey = keystoreProps["storeFile"] != null

android {
    namespace = "com.tuigo"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.tuigo"
        minSdk = 26
        targetSdk = 36
        versionCode = 9
        versionName = "0.0.9"
        testInstrumentationRunner = "com.tuigo.MobileSmokeTest"
    }

    signingConfigs {
        if (hasReleaseKey) {
            create("release") {
                storeFile = rootProject.file(keystoreProps["storeFile"] as String)
                storePassword = keystoreProps["storePassword"] as String
                keyAlias = keystoreProps["keyAlias"] as String
                keyPassword = keystoreProps["keyPassword"] as String
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            if (hasReleaseKey) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

kotlin {
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
    }
}

dependencies {
    implementation(project(":engine"))
}
