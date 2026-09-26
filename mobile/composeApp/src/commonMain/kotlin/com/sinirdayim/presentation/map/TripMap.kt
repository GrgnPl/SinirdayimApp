package com.sinirdayim.presentation.map

import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import com.sinirdayim.domain.model.StepKind
import com.sinirdayim.domain.model.TripPlan
import com.sinirdayim.presentation.theme.LocalLevelColors
import org.maplibre.compose.camera.CameraPosition
import org.maplibre.compose.expressions.dsl.const
import org.maplibre.compose.layers.CircleLayer
import org.maplibre.compose.layers.LineLayer
import org.maplibre.compose.map.MaplibreMap
import org.maplibre.compose.map.rememberMapState
import org.maplibre.compose.sources.rememberGeoJsonSource
import org.maplibre.compose.util.DpPadding
import org.maplibre.compose.util.MaplibreComposable
import org.maplibre.spatialk.geojson.Position

/** Route line with border crossings, breaks and daily rests. */
@Composable
fun TripMap(plan: TripPlan, modifier: Modifier = Modifier) {
    val levels = LocalLevelColors.current
    val routeColor = MaterialTheme.colorScheme.primary
    val surface = MaterialTheme.colorScheme.surface
    val start = plan.route.firstOrNull()

    val state = rememberMapState(
        baseStyle = appMapStyle(),
        initialCameraPosition = CameraPosition(
            target = Position(start?.lng ?: 35.0, start?.lat ?: 39.0),
            zoom = 5.0,
        ),
    ) {
        val route = rememberGeoJsonSource(lineData(plan.route))
        LineLayer(id = "route-casing", source = route, color = const(surface), width = const(7.dp))
        LineLayer(id = "route", source = route, color = const(routeColor), width = const(4.dp))

        stopLayer("breaks", plan, StepKind.BREAK, levels.medium, surface)
        stopLayer("rests", plan, StepKind.DAILY_REST, RestColor, surface)

        val crossings = rememberGeoJsonSource(pointsData(plan.crossings.map { PointFeature(it.location) }))
        CircleLayer(
            id = "crossings", source = crossings,
            color = const(levels.high), radius = const(7.dp),
            strokeColor = const(surface), strokeWidth = const(2.dp),
        )

        val ends = rememberGeoJsonSource(pointsData(listOfNotNull(plan.route.firstOrNull(), plan.route.lastOrNull()).map { PointFeature(it) }))
        CircleLayer(
            id = "ends", source = ends,
            color = const(routeColor), radius = const(6.dp),
            strokeColor = const(surface), strokeWidth = const(3.dp),
        )
    }

    LaunchedEffect(plan) {
        boundsOf(plan.route)?.let { state.fitCameraToBounds(it, fitPadding = DpPadding(24.dp, 24.dp, 24.dp, 24.dp)) }
    }

    MaplibreMap(modifier = modifier, state = state)
}

@Composable
@MaplibreComposable
private fun stopLayer(id: String, plan: TripPlan, kind: StepKind, color: Color, stroke: Color) {
    val source = rememberGeoJsonSource(pointsData(plan.steps.filter { it.kind == kind }.map { PointFeature(it.location) }))
    CircleLayer(
        id = id, source = source,
        color = const(color), radius = const(6.dp),
        strokeColor = const(stroke), strokeWidth = const(2.dp),
    )
}

val RestColor = Color(0xFF5B6CFF)
