@file:OptIn(ExperimentalTime::class)

package com.sinirbekleme.data.remote

import com.sinirbekleme.domain.model.GeoPoint
import com.sinirbekleme.domain.model.Place
import com.sinirbekleme.domain.model.StepKind
import com.sinirbekleme.domain.model.StopReason
import com.sinirbekleme.domain.model.TripAlternative
import com.sinirbekleme.domain.model.TripCrossing
import com.sinirbekleme.domain.model.TripPlan
import com.sinirbekleme.domain.model.TripRequest
import com.sinirbekleme.domain.model.TripStep
import com.sinirbekleme.domain.model.TripTotals
import kotlin.time.ExperimentalTime
import kotlin.time.Instant

fun PlaceDto.toDomain() = Place(name, label, country, GeoPoint(location.lat, location.lng))

fun TripRequest.toDto() = PlanRequestDto(
    origin = GeoPointDto(origin.lat, origin.lng),
    destination = GeoPointDto(destination.lat, destination.lng),
    departAt = departAt.toString(),
    driver = DriverDto(driver.continuousDrivingMin, driver.dailyDrivingMin, driver.extendedDaysLeft),
)

fun TripPlanDto.toDomain() = TripPlan(
    distanceKm = distanceKm,
    departure = Instant.parse(departure),
    arrival = Instant.parse(arrival),
    totals = TripTotals(totals.drivingMin, totals.breakMin, totals.dailyRestMin, totals.borderWaitMin, totals.totalMin),
    crossings = crossings.map {
        val est = it.estimate.toDomain()
        TripCrossing(it.id, it.name, it.from, it.to, it.atKm, est.waitMinutes, est.level)
    },
    steps = steps.mapNotNull { it.toDomainOrNull() },
    allWaitsKnown = allWaitsKnown,
    alternatives = alternatives.map {
        TripAlternative(it.via, it.distanceKm, it.totalMin, it.borderWaitMin, it.allWaitsKnown, it.selected)
    },
    polyline = polyline,
)

private fun TripStepDto.toDomainOrNull(): TripStep? {
    val kind = when (kind) {
        "drive" -> StepKind.DRIVE
        "break" -> StepKind.BREAK
        "daily_rest" -> StepKind.DAILY_REST
        "border_wait" -> StepKind.BORDER_WAIT
        else -> return null
    }
    return TripStep(
        kind = kind,
        start = Instant.parse(start),
        end = Instant.parse(end),
        durationMin = durationMin,
        fromKm = fromKm,
        toKm = toKm,
        location = GeoPoint(location.lat, location.lng),
        reason = when (reason) {
            "continuous_driving_limit" -> StopReason.CONTINUOUS_DRIVING
            "daily_driving_limit" -> StopReason.DAILY_DRIVING
            "duty_period_limit" -> StopReason.DUTY_PERIOD
            else -> null
        },
        crossingId = crossingId,
    )
}
