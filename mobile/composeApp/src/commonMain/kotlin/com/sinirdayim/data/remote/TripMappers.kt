@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.data.remote

import com.sinirdayim.domain.model.GeoPoint
import com.sinirdayim.domain.model.AppointmentPlan
import com.sinirdayim.domain.model.Place
import com.sinirdayim.domain.model.Procedure
import com.sinirdayim.domain.model.ProcedureKind
import com.sinirdayim.domain.model.RestArea
import com.sinirdayim.domain.model.RestAreaKind
import com.sinirdayim.domain.model.StepKind
import com.sinirdayim.domain.model.StopReason
import com.sinirdayim.domain.model.TripAlternative
import com.sinirdayim.domain.model.TripCrossing
import com.sinirdayim.domain.model.TripPlan
import com.sinirdayim.domain.model.TripRequest
import com.sinirdayim.domain.model.TripStep
import com.sinirdayim.domain.model.TripTotals
import kotlin.time.ExperimentalTime
import kotlin.time.Instant

fun PlaceDto.toDomain() = Place(name, label, country, GeoPoint(location.lat, location.lng))

fun TripRequest.toDto() = PlanRequestDto(
    origin = GeoPointDto(origin.lat, origin.lng),
    destination = GeoPointDto(destination.lat, destination.lng),
    departAt = departAt.toString(),
    driver = DriverDto(
        continuousDrivingMin = driver.continuousDrivingMin,
        splitBreakTaken = driver.splitBreakTaken,
        dailyDrivingMin = driver.dailyDrivingMin,
        extendedDaysLeft = driver.extendedDaysLeft,
        reducedRestsLeft = driver.reducedRestsLeft,
        weeklyDrivingMin = driver.weeklyDrivingMin,
        prevWeekDrivingMin = driver.prevWeekDrivingMin,
    ),
    appointment = appointment?.let { AppointmentDto(it.crossingId, it.at.toString()) },
)

fun TripPlanDto.toDomain() = TripPlan(
    distanceKm = distanceKm,
    departure = Instant.parse(departure),
    arrival = Instant.parse(arrival),
    totals = TripTotals(
        drivingMin = totals.drivingMin,
        breakMin = totals.breakMin,
        dailyRestMin = totals.dailyRestMin,
        weeklyRestMin = totals.weeklyRestMin,
        borderWaitMin = totals.borderWaitMin,
        totalMin = totals.totalMin,
    ),
    crossings = crossings.map {
        val est = it.estimate.toDomain()
        TripCrossing(
            id = it.id,
            name = it.name,
            location = GeoPoint(it.location.lat, it.location.lng),
            from = it.from,
            to = it.to,
            atKm = it.atKm,
            waitMinutes = est.waitMinutes,
            level = est.level,
            procedures = it.procedures.mapNotNull { p -> p.toDomainOrNull() },
            suggestedAppointment = it.suggestedAppointment?.let(::parseInstantOrNull),
        )
    },
    steps = steps.mapNotNull { it.toDomainOrNull() },
    allWaitsKnown = allWaitsKnown,
    alternatives = alternatives.map {
        TripAlternative(it.via, it.distanceKm, it.totalMin, it.borderWaitMin, it.allWaitsKnown, it.selected)
    },
    route = decodePolyline(polyline),
    appointment = appointment?.let {
        AppointmentPlan(
            crossingId = it.crossingId,
            crossingName = it.crossingName,
            at = Instant.parse(it.at),
            arriveAt = Instant.parse(it.arriveAt),
            slackMin = it.slackMin,
            onTime = it.onTime,
            latestDeparture = it.latestDeparture?.let(::parseInstantOrNull),
        )
    },
)

private fun ProcedureDto.toDomainOrNull(): Procedure? = Procedure(
    kind = when (kind) {
        "appointment" -> ProcedureKind.APPOINTMENT
        "truck_park" -> ProcedureKind.TRUCK_PARK
        else -> return null
    },
    system = system,
    url = url,
    mandatory = mandatory,
    fee = fee,
    note = note,
)

private fun parseInstantOrNull(value: String): Instant? = runCatching { Instant.parse(value) }.getOrNull()

private fun stepKind(value: String?): StepKind? = when (value) {
    "drive" -> StepKind.DRIVE
    "break" -> StepKind.BREAK
    "daily_rest" -> StepKind.DAILY_REST
    "weekly_rest" -> StepKind.WEEKLY_REST
    "border_wait" -> StepKind.BORDER_WAIT
    else -> null
}

private fun TripStepDto.toDomainOrNull(): TripStep? {
    val kind = stepKind(kind) ?: return null
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
            "weekly_driving_limit" -> StopReason.WEEKLY_DRIVING
            "weekly_rest_due" -> StopReason.WEEKLY_REST_DUE
            else -> null
        },
        crossingId = crossingId,
        restArea = restArea?.toDomain(),
        reduced = reduced,
        countsAs = stepKind(countsAs),
    )
}

private fun RestAreaDto.toDomain() = RestArea(
    id = id,
    name = name?.takeIf { it.isNotBlank() },
    kind = when (kind) {
        "services" -> RestAreaKind.SERVICES
        "truck_parking" -> RestAreaKind.TRUCK_PARKING
        "truck_fuel" -> RestAreaKind.TRUCK_FUEL
        else -> RestAreaKind.REST_AREA
    },
    location = GeoPoint(location.lat, location.lng),
    toilets = toilets,
    shower = shower,
    restaurant = restaurant,
    fee = fee,
    supervised = supervised,
    hgvCapacity = hgvCapacity,
)
