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
        applicationId = "com.tuigo.tiengviet"
        minSdk = 26
        targetSdk = 36
        versionCode = 15
        versionName = providers.gradleProperty("tuigoVersion").orElse("0.1.5").get()
        testInstrumentationRunner = "com.tuigo.MobileSmokeTest"
    }

    signingConfigs {
        if (hasReleaseKey) {
            create("release") {
                storeFile = rootProject.file(keystoreProps["storeFile"] as String)
                storePassword = keystoreProps["storePassword"] as String
                keyAlias = keystoreProps["keyAlias"] as String
                keyPassword = keystoreProps["keyPassword"] as String
                enableV1Signing = true
                enableV2Signing = true
                enableV3Signing = true
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
    // Inline-autofill suggestion styling (chips rendered by autofill providers).
    implementation("androidx.autofill:autofill:1.3.0")
    implementation("com.google.android.flexbox:flexbox:3.0.0")
    // FileProvider for re-pasting clipboard-history images via commitContent.
    implementation("androidx.core:core:1.13.1")
}
