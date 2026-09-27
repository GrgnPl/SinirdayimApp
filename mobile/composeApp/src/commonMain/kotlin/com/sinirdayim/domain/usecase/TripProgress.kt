@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.domain.usecase

import com.sinirdayim.domain.model.GeoPoint
import com.sinirdayim.domain.model.StepKind
import com.sinirdayim.domain.model.TripPlan
import kotlin.math.PI
import kotlin.math.abs
import kotlin.math.asin
import kotlin.math.cos
import kotlin.math.hypot
import kotlin.math.sin
import kotlin.math.sqrt
import kotlin.time.Duration
import kotlin.time.Duration.Companion.ZERO
import kotlin.time.ExperimentalTime
import kotlin.time.Instant

/** Where the truck is compared with the plan. */
data class TripProgress(
    val km: Double,
    val totalKm: Double,
    /** Distance from the route line; large values mean the truck left the route. */
    val offRouteKm: Double,
    /** Positive when behind the plan, negative when ahead. */
    val delay: Duration,
    val projectedArrival: Instant,
    val appointment: AppointmentProgress?,
) {
    val offRoute: Boolean get() = offRouteKm > OFF_ROUTE_KM

    companion object {
        const val OFF_ROUTE_KM = 2.0
    }
}

data class AppointmentProgress(
    val crossingName: String,
    val slot: Instant,
    val projectedGateArrival: Instant,
    val passed: Boolean,
) {
    /** How late the truck will reach the gate; zero or negative is on time. */
    val lateBy: Duration get() = if (passed) ZERO else projectedGateArrival - slot
}

/**
 * Compares a GPS fix with the plan. The route and the timeline are the plan's
 * own; nothing is re-planned, so this works offline.
 */
class TripTracker(val plan: TripPlan) {
    private val cum: DoubleArray = cumulative(plan.route)

    fun progress(position: GeoPoint, now: Instant): TripProgress {
        val (km, offset) = project(position)
        val delay = delayAt(km, now)
        val late = if (delay.isPositive()) delay else ZERO
        val appointment = plan.appointment?.let { a ->
            val gateKm = plan.crossings.firstOrNull { it.id == a.crossingId }?.atKm
            AppointmentProgress(
                crossingName = a.crossingName,
                slot = a.at,
                projectedGateArrival = a.arriveAt + late,
                passed = gateKm != null && km > gateKm + GATE_PASSED_KM,
            )
        }
        return TripProgress(
            km = km,
            totalKm = cum.lastOrNull() ?: 0.0,
            offRouteKm = offset,
            delay = delay,
            projectedArrival = plan.arrival + late,
            appointment = appointment,
        )
    }

    /**
     * Delay against the plan at km: the plan reaches km at some time and may
     * stop there for a while; being anywhere in that window is on time.
     */
    internal fun delayAt(km: Double, now: Instant): Duration {
        var arrive: Instant? = null
        var leave: Instant? = null
        for (s in plan.steps) {
            if (s.kind == StepKind.DRIVE) {
                if (arrive == null && km <= s.toKm) {
                    val span = s.toKm - s.fromKm
                    val f = if (span > 0) ((km - s.fromKm) / span).coerceIn(0.0, 1.0) else 1.0
                    arrive = s.start + (s.end - s.start) * f
                    leave = arrive
                } else if (arrive != null) {
                    break
                }
            } else if (arrive != null) {
                // A stop right at km extends the window; one further on ends the search.
                if (abs(s.fromKm - km) > STOP_TOLERANCE_KM) break
                leave = s.end
            }
        }
        val a = arrive ?: return now - plan.arrival
        val l = leave ?: a
        return when {
            now < a -> now - a
            now <= l -> ZERO
            else -> now - l
        }
    }

    /** Projection onto the route line: km along the route and offset in km. */
    internal fun project(p: GeoPoint): Pair<Double, Double> {
        val r = plan.route
        if (r.size < 2) return 0.0 to Double.MAX_VALUE
        var bestKm = 0.0
        var bestOffset = Double.MAX_VALUE
        for (i in 0 until r.size - 1) {
            val (t, d) = projectOnSegment(r[i], r[i + 1], p)
            if (d < bestOffset) {
                bestOffset = d
                bestKm = cum[i] + t * (cum[i + 1] - cum[i])
            }
        }
        return bestKm to bestOffset
    }

    private companion object {
        const val GATE_PASSED_KM = 1.0
        const val STOP_TOLERANCE_KM = 0.5
    }
}

private fun cumulative(route: List<GeoPoint>): DoubleArray {
    val out = DoubleArray(route.size)
    for (i in 1 until route.size) out[i] = out[i - 1] + distanceKm(route[i - 1], route[i])
    return out
}

private fun distanceKm(a: GeoPoint, b: GeoPoint): Double {
    val la1 = a.lat.rad()
    val la2 = b.lat.rad()
    val h = sin((la2 - la1) / 2).let { it * it } + cos(la1) * cos(la2) * sin((b.lng - a.lng).rad() / 2).let { it * it }
    return 2 * 6371.0 * asin(sqrt(h))
}

/** Local flat projection, fine for route segments of a few km. */
private fun projectOnSegment(a: GeoPoint, b: GeoPoint, p: GeoPoint): Pair<Double, Double> {
    val kx = cos(a.lat.rad()) * 111.32
    val ky = 110.57
    val bx = (b.lng - a.lng) * kx
    val by = (b.lat - a.lat) * ky
    val px = (p.lng - a.lng) * kx
    val py = (p.lat - a.lat) * ky
    val l2 = bx * bx + by * by
    val t = if (l2 > 0) ((px * bx + py * by) / l2).coerceIn(0.0, 1.0) else 0.0
    return t to hypot(px - t * bx, py - t * by)
}

private fun Double.rad() = this * PI / 180
