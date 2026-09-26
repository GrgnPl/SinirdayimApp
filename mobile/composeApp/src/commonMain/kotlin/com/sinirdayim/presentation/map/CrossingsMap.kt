package com.sinirdayim.presentation.map

import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.sinirdayim.domain.model.CrossingStatus
import com.sinirdayim.domain.model.Direction
import com.sinirdayim.domain.model.Level
import com.sinirdayim.presentation.theme.LocalLevelColors
import kotlinx.serialization.json.jsonPrimitive
import org.maplibre.compose.camera.CameraPosition
import org.maplibre.compose.expressions.dsl.const
import org.maplibre.compose.interaction.ClickResult
import org.maplibre.compose.layers.CircleLayer
import org.maplibre.compose.map.MaplibreMap
import org.maplibre.compose.map.rememberMapState
import org.maplibre.compose.sources.rememberGeoJsonSource
import org.maplibre.compose.util.DpPadding
import org.maplibre.compose.util.MaplibreComposable
import org.maplibre.spatialk.geojson.Position

/** Crossings as dots colored by traffic level; tapping one calls [onSelect]. */
@Composable
fun CrossingsMap(
    crossings: List<CrossingStatus>,
    direction: Direction,
    onSelect: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalLevelColors.current
    val stroke = MaterialTheme.colorScheme.surface

    val state = rememberMapState(
        baseStyle = appMapStyle(),
        initialCameraPosition = CameraPosition(target = Position(33.0, 40.5), zoom = 4.2),
    ) {
        // One source per level keeps styling free of data-driven expressions.
        Level.entries.forEach { level ->
            levelLayer(level, crossings.filter { it.estimate(direction).level == level }, colors.of(level), stroke, onSelect)
        }
    }

    LaunchedEffect(Unit) {
        boundsOf(crossings.map { it.crossing.location })?.let {
            state.fitCameraToBounds(it, fitPadding = DpPadding(48.dp, 120.dp, 48.dp, 160.dp))
        }
    }

    MaplibreMap(modifier = modifier, state = state)
}

@Composable
@MaplibreComposable
private fun levelLayer(
    level: Level,
    items: List<CrossingStatus>,
    color: androidx.compose.ui.graphics.Color,
    stroke: androidx.compose.ui.graphics.Color,
    onSelect: (String) -> Unit,
) {
    val source = rememberGeoJsonSource(
        pointsData(items.map { PointFeature(it.crossing.location, mapOf("id" to it.crossing.id)) }),
    )
    CircleLayer(
        id = "crossings-${level.name.lowercase()}",
        source = source,
        color = const(color),
        radius = const(if (level == Level.UNKNOWN) 6.dp else 9.dp),
        strokeColor = const(stroke),
        strokeWidth = const(3.dp),
        onClick = { features ->
            val id = features.firstOrNull()?.properties?.get("id")?.jsonPrimitive?.content
            if (id != null) {
                onSelect(id)
                ClickResult.Consume
            } else {
                ClickResult.Pass
            }
        },
    )
}
