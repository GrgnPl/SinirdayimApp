package com.sinirbekleme.domain.model

import kotlin.time.ExperimentalTime
import kotlin.time.Instant

enum class Direction { EXPORT, IMPORT }

enum class Level { LOW, MEDIUM, HIGH, UNKNOWN }

enum class Confidence { HIGH, MEDIUM, LOW }

data class GeoPoint(val lat: Double, val lng: Double)

data class Crossing(
    val id: String,
    val name: String,
    /** ISO-3166 alpha-2; export flows from [countries].first to [countries].second. */
    val countries: Pair<String, String>,
    val location: GeoPoint,
)

@OptIn(ExperimentalTime::class)
data class WaitEstimate(
    val vehicles: Int?,
    val waitMinutes: Int?,
    val level: Level,
    val confidence: Confidence,
    val method: String,
    val sources: List<String>,
    val dataAt: Instant?,
)

data class CrossingStatus(
    val crossing: Crossing,
    val export: WaitEstimate,
    val import: WaitEstimate,
) {
    fun estimate(direction: Direction): WaitEstimate =
        if (direction == Direction.EXPORT) export else import
}

@OptIn(ExperimentalTime::class)
data class Snapshot(
    val direction: Direction,
    val source: String,
    val observedAt: Instant,
    val queueKm: Double?,
    val queueVehicles: Int?,
    val parkedVehicles: Int?,
    val dailyThroughput: Int?,
    val waitMinutes: Int?,
)

data class CrossingDetail(
    val status: CrossingStatus,
    val history: List<Snapshot>,
)
