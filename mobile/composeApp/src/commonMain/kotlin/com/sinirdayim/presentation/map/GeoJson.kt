package com.sinirdayim.presentation.map

import com.sinirdayim.domain.model.GeoPoint
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonObject
import org.maplibre.compose.sources.GeoJsonData
import org.maplibre.spatialk.geojson.BoundingBox

/** A point feature with string properties (e.g. an id for click handling). */
data class PointFeature(val at: GeoPoint, val properties: Map<String, String> = emptyMap())

fun pointsData(points: List<PointFeature>): GeoJsonData =
    collection(points.map { feature(geometry("Point", position(it.at)), it.properties) })

fun lineData(line: List<GeoPoint>): GeoJsonData =
    collection(
        if (line.size < 2) emptyList()
        else listOf(feature(geometry("LineString", JsonArray(line.map(::position))), emptyMap())),
    )

/** Bounds of the given points, padded slightly so markers are not on the edge. */
fun boundsOf(points: List<GeoPoint>): BoundingBox? {
    if (points.isEmpty()) return null
    val pad = 0.02
    return BoundingBox(
        west = points.minOf { it.lng } - pad,
        south = points.minOf { it.lat } - pad,
        east = points.maxOf { it.lng } + pad,
        north = points.maxOf { it.lat } + pad,
    )
}

/** GeoJSON positions are [longitude, latitude]. */
private fun position(p: GeoPoint) = JsonArray(listOf(JsonPrimitive(p.lng), JsonPrimitive(p.lat)))

private fun geometry(type: String, coordinates: JsonElement) = buildJsonObject {
    put("type", type)
    put("coordinates", coordinates)
}

private fun feature(geometry: JsonObject, properties: Map<String, String>) = buildJsonObject {
    put("type", "Feature")
    put("geometry", geometry)
    putJsonObject("properties") { properties.forEach { (k, v) -> put(k, v) } }
}

private fun collection(features: List<JsonObject>): GeoJsonData = GeoJsonData.JsonString(
    buildJsonObject {
        put("type", "FeatureCollection")
        put("features", JsonArray(features))
    }.toString(),
)
