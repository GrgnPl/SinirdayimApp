// Android application shell: entry point, manifest and build-time config.
plugins {
    alias(libs.plugins.androidApplication)
    alias(libs.plugins.composeMultiplatform)
    alias(libs.plugins.composeCompiler)
}

android {
    namespace = "com.sinirdayim.android"
    compileSdk = libs.versions.android.compileSdk.get().toInt()

    defaultConfig {
        applicationId = "com.sinirdayim"
        minSdk = libs.versions.android.minSdk.get().toInt()
        targetSdk = libs.versions.android.targetSdk.get().toInt()
        versionCode = 1
        versionName = "0.1.0"
    }
    buildFeatures { buildConfig = true }
    buildTypes {
        debug {
            // Emulator reaches the host machine through 10.0.2.2. Override with -PapiBaseUrl=...
            val apiBaseUrl = providers.gradleProperty("apiBaseUrl").getOrElse("http://10.0.2.2:8080")
            buildConfigField("String", "API_BASE_URL", "\"$apiBaseUrl\"")
            manifestPlaceholders["usesCleartextTraffic"] = "true"
        }
        release {
            isMinifyEnabled = false
            manifestPlaceholders["usesCleartextTraffic"] = "false"
            buildConfigField("String", "API_BASE_URL", "\"https://api.sinirdayim.com\"")
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    implementation(projects.composeApp)
    implementation(libs.androidx.activity.compose)
    runtimeOnly(libs.maplibre.compose.runtime.opengl.android)
}
