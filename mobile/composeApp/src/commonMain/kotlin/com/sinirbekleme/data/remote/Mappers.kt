@file:OptIn(ExperimentalTime::class)

package com.sinirbekleme.data.remote

import com.sinirbekleme.domain.model.Confidence
import com.sinirbekleme.domain.model.Crossing
import com.sinirbekleme.domain.model.CrossingDetail
import com.sinirbekleme.domain.model.CrossingStatus
import com.sinirbekleme.domain.model.Direction
import com.sinirbekleme.domain.model.GeoPoint
import com.sinirbekleme.domain.model.Level
import com.sinirbekleme.domain.model.Snapshot
import com.sinirbekleme.domain.model.WaitEstimate
import kotlin.time.ExperimentalTime
import kotlin.time.Instant

fun CrossingStatusDto.toDomain() = CrossingStatus(
    crossing = crossing(id, name, countries, location),
    export = export.toDomain(),
    import = importEstimate.toDomain(),
)

fun CrossingDetailDto.toDomain() = CrossingDetail(
    status = CrossingStatus(
        crossing = crossing(id, name, countries, location),
        export = export.toDomain(),
        import = importEstimate.toDomain(),
    ),
    history = history.mapNotNull { it.toDomainOrNull() },
)

private fun crossing(id: String, name: String, countries: List<String>, location: GeoPointDto) = Crossing(
    id = id,
    name = name,
    countries = (countries.getOrNull(0) ?: "") to (countries.getOrNull(1) ?: ""),
    location = GeoPoint(location.lat, location.lng),
)

fun WaitEstimateDto.toDomain() = WaitEstimate(
    vehicles = vehicles,
    waitMinutes = waitMinutes,
    level = when (level) {
        "low" -> Level.LOW
        "medium" -> Level.MEDIUM
        "high" -> Level.HIGH
        else -> Level.UNKNOWN
    },
    confidence = when (confidence) {
        "high" -> Confidence.HIGH
        "medium" -> Confidence.MEDIUM
        else -> Confidence.LOW
    },
    method = method,
    sources = basedOn,
    // The backend sends Go's zero time when there is no data.
    dataAt = dataAt?.let(::parseInstant)?.takeIf { it.epochSeconds > 0 },
)

fun SnapshotDto.toDomainOrNull(): Snapshot? {
    val at = parseInstant(observedAt) ?: return null
    return Snapshot(
        direction = if (direction == "import") Direction.IMPORT else Direction.EXPORT,
        source = source,
        observedAt = at,
        queueKm = queueKm,
        queueVehicles = queueVehicles,
        parkedVehicles = parkedVehicles,
        dailyThroughput = dailyThroughput,
        waitMinutes = waitMinutes,
    )
}

private fun parseInstant(value: String): Instant? = runCatching { Instant.parse(value) }.getOrNull()
