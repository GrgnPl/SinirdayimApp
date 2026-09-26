package com.sinirdayim.presentation.map

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import org.maplibre.compose.style.BaseStyle

// OpenFreeMap: free OSM vector tiles, no API key. To be replaced by our own
// tile server with a truck-focused style.
private val Light = BaseStyle.Uri("https://tiles.openfreemap.org/styles/positron")
private val Dark = BaseStyle.Uri("https://tiles.openfreemap.org/styles/dark")

@Composable
fun appMapStyle(): BaseStyle = if (isSystemInDarkTheme()) Dark else Light
