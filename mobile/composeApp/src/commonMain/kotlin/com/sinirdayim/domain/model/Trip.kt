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
    val splitBreakTaken: Boolean = false,
    val dailyDrivingMin: Int = 0,
    val extendedDaysLeft: Int = 2,
    val reducedRestsLeft: Int = 0,
    val weeklyDrivingMin: Int = 0,
    val prevWeekDrivingMin: Int = 0,
)

data class TripRequest(
    val origin: GeoPoint,
    val destination: GeoPoint,
    val departAt: Instant,
    val driver: DriverState,
    val appointment: Appointment? = null,
)

/** A booked slot at a crossing, e.g. RSS at Kapıkule. */
data class Appointment(val crossingId: String, val at: Instant)

enum class ProcedureKind { APPOINTMENT, TRUCK_PARK }

/** A formality at a crossing in the travel direction. */
data class Procedure(
    val kind: ProcedureKind,
    val system: String,
    val url: String?,
    val mandatory: Boolean,
    val fee: String?,
    val note: String?,
)

/** How the trip fits a booked slot. */
data class AppointmentPlan(
    val crossingId: String,
    val crossingName: String,
    val at: Instant,
    val arriveAt: Instant,
    /** Minutes between arrival and the slot; negative when late. */
    val slackMin: Int,
    val onTime: Boolean,
    /** Latest departure that still makes the slot; null if already too late. */
    val latestDeparture: Instant?,
)

enum class StepKind { DRIVE, BREAK, DAILY_REST, WEEKLY_REST, BORDER_WAIT }

enum class StopReason { CONTINUOUS_DRIVING, DAILY_DRIVING, DUTY_PERIOD, WEEKLY_DRIVING, WEEKLY_REST_DUE }

enum class RestAreaKind { SERVICES, REST_AREA, TRUCK_PARKING, TRUCK_FUEL }

/** A place to stop. Facility flags are null when unknown. */
data class RestArea(
    val id: String,
    val name: String?,
    val kind: RestAreaKind,
    val location: GeoPoint,
    val toilets: Boolean?,
    val shower: Boolean?,
    val restaurant: Boolean?,
    val fee: Boolean?,
    val supervised: Boolean?,
    val hgvCapacity: Int?,
)

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
    val restArea: RestArea?,
    /** A 9h daily rest, or the 30 min second part of a split break. */
    val reduced: Boolean,
    /** For border waits: the rest the wait counted as, if any. */
    val countsAs: StepKind?,
)

data class TripCrossing(
    val id: String,
    val name: String,
    val location: GeoPoint,
    val from: String,
    val to: String,
    val atKm: Double,
    val waitMinutes: Int?,
    val level: Level,
    val procedures: List<Procedure>,
    val suggestedAppointment: Instant?,
) {
    val appointmentSystem: Procedure? get() = procedures.firstOrNull { it.kind == ProcedureKind.APPOINTMENT }
}

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
    val weeklyRestMin: Int,
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
    val route: List<GeoPoint>,
    val appointment: AppointmentPlan?,
)
