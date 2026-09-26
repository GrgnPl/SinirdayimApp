@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.domain.model

import kotlin.time.ExperimentalTime
import kotlin.time.Instant

data class Place(
    val name: String,
    val label: String,
    val country: String,
    val location: GeoPoint,
)

/** Driver's tachograph state at departure. */
data class DriverState(
    val continuousDrivingMin: Int = 0,
    val dailyDrivingMin: Int = 0,
    val extendedDaysLeft: Int = 2,
)

data class TripRequest(
    val origin: GeoPoint,
    val destination: GeoPoint,
    val departAt: Instant,
    val driver: DriverState,
)

enum class StepKind { DRIVE, BREAK, DAILY_REST, BORDER_WAIT }

enum class StopReason { CONTINUOUS_DRIVING, DAILY_DRIVING, DUTY_PERIOD }

data class TripStep(
    val kind: StepKind,
    val start: Instant,
    val end: Instant,
    val durationMin: Int,
    val fromKm: Double,
    val toKm: Double,
    val location: GeoPoint,
    val reason: StopReason?,
    val crossingId: String?,
)

data class TripCrossing(
    val id: String,
    val name: String,
    val from: String,
    val to: String,
    val atKm: Double,
    val waitMinutes: Int?,
    val level: Level,
)

data class TripAlternative(
    val via: List<String>,
    val distanceKm: Double,
    val totalMin: Int,
    val borderWaitMin: Int,
    val allWaitsKnown: Boolean,
    val selected: Boolean,
)

data class TripTotals(
    val drivingMin: Int,
    val breakMin: Int,
    val dailyRestMin: Int,
    val borderWaitMin: Int,
    val totalMin: Int,
)

data class TripPlan(
    val distanceKm: Double,
    val departure: Instant,
    val arrival: Instant,
    val totals: TripTotals,
    val crossings: List<TripCrossing>,
    val steps: List<TripStep>,
    val allWaitsKnown: Boolean,
    val alternatives: List<TripAlternative>,
    val polyline: String,
)
