@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.domain.usecase

import com.sinirdayim.domain.model.AppointmentPlan
import com.sinirdayim.domain.model.GeoPoint
import com.sinirdayim.domain.model.Level
import com.sinirdayim.domain.model.StepKind
import com.sinirdayim.domain.model.TripCrossing
import com.sinirdayim.domain.model.TripPlan
import com.sinirdayim.domain.model.TripStep
import com.sinirdayim.domain.model.TripTotals
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNotNull
import kotlin.test.assertTrue
import kotlin.time.Duration
import kotlin.time.Duration.Companion.hours
import kotlin.time.Duration.Companion.minutes
import kotlin.time.ExperimentalTime
import kotlin.time.Instant

class TripTrackerTest {
    // A straight road north, 1° of latitude ≈ 111.2 km; driven at 60 km/h.
    private fun at(km: Double) = GeoPoint(40.0 + km / 111.195, 30.0)
    private val t0 = Instant.parse("2026-09-28T06:00:00Z")

    private fun step(kind: StepKind, start: Duration, length: Duration, fromKm: Double, toKm: Double) =
        TripStep(kind, t0 + start, t0 + start + length, length.inWholeMinutes.toInt(), fromKm, toKm, at(fromKm), null, null, null, false, null)

    // Drive 0→60 km (1 h), gate wait at km 60 until 09:00, drive 60→120 km (1 h).
    private val plan = TripPlan(
        distanceKm = 120.0,
        departure = t0,
        arrival = t0 + 4.hours,
        totals = TripTotals(120, 0, 0, 0, 120, 240),
        crossings = listOf(TripCrossing("gate", "Kapı", at(60.0), "TR", "BG", 60.0, null, Level.UNKNOWN, emptyList(), null)),
        steps = listOf(
            step(StepKind.DRIVE, 0.hours, 1.hours, 0.0, 60.0),
            step(StepKind.BORDER_WAIT, 1.hours, 2.hours, 60.0, 60.0),
            step(StepKind.DRIVE, 3.hours, 1.hours, 60.0, 120.0),
        ),
        allWaitsKnown = true,
        alternatives = emptyList(),
        route = (0..120 step 5).map { at(it.toDouble()) },
        appointment = AppointmentPlan("gate", "Kapı", t0 + 2.hours, t0 + 1.hours, 60, true, t0, null),
    )
    private val tracker = TripTracker(plan)

    @Test
    fun onScheduleMidDrive() {
        val p = tracker.progress(at(30.0), t0 + 30.minutes)
        assertEquals(30.0, p.km, 0.5)
        assertTrue(p.delay.absoluteValue < 2.minutes, "delay ${p.delay}")
        assertFalse(p.offRoute)
    }

    @Test
    fun behindPlanMakesAppointmentLate() {
        // Planned at km 30 at 06:30; being there at 08:00 is 90 min behind.
        val p = tracker.progress(at(30.0), t0 + 2.hours)
        assertTrue((p.delay - 90.minutes).absoluteValue < 2.minutes, "delay ${p.delay}")
        val a = assertNotNull(p.appointment)
        // Gate planned at 07:00 + 90 min = 08:30; slot 08:00 → 30 min late.
        assertTrue((a.lateBy - 30.minutes).absoluteValue < 2.minutes, "late by ${a.lateBy}")
        assertEquals(t0 + 4.hours + p.delay, p.projectedArrival)
    }

    @Test
    fun waitingAtTheGateIsOnTime() {
        val p = tracker.progress(at(60.0), t0 + 2.hours + 30.minutes)
        assertEquals(Duration.ZERO, p.delay)
    }

    @Test
    fun aheadOfPlanIsNegative() {
        val p = tracker.progress(at(90.0), t0 + 3.hours)
        assertTrue(p.delay < -20.minutes, "delay ${p.delay}")
        assertTrue(assertNotNull(p.appointment).passed)
    }

    @Test
    fun farFromRouteIsOffRoute() {
        val p = tracker.progress(GeoPoint(40.3, 30.2), t0)
        assertTrue(p.offRoute, "offset ${p.offRouteKm}")
    }
}
