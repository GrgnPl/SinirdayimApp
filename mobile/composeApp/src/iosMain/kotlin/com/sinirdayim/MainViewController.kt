package com.sinirdayim

import androidx.compose.ui.window.ComposeUIViewController

// The simulator reaches the host machine through localhost.
fun MainViewController(apiBaseUrl: String = "http://localhost:8080") =
    ComposeUIViewController { App(apiBaseUrl) }
