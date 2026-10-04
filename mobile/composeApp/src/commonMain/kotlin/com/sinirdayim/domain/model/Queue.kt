@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.domain.model

import kotlin.time.ExperimentalTime
import kotlin.time.Instant

enum class TrendDirection { GROWING, SHRINKING, STABLE }

data class QueueTrend(val vehiclesPerHour: Double, val direction: TrendDirection, val to: Instant)

/** Expected wait for a truck reaching the gate at [at]. */
data class HourOutlook(val at: Instant, val vehicles: Int, val waitMinutes: Int, val level: Level)

enum class ReportKind { IN_QUEUE, PASSED }

data class DriverReport(
    val kind: ReportKind,
    val at: Instant,
    val vehiclesAhead: Int?,
    val queueKm: Double?,
    val waitMinutes: Int?,
    val note: String?,
)

/** What one source last said; [reports] is set for driver reports. */
data class SourceStatus(val source: String, val observedAt: Instant, val reports: Int)

data class DirectionQueue(
    val direction: Direction,
    /** Official data blended with recent driver reports. */
    val estimate: WaitEstimate,
    /** Official data only. */
    val official: WaitEstimate,
    val dailyThroughput: Int?,
    val trend: QueueTrend?,
    val outlook: List<HourOutlook>,
    val sources: List<SourceStatus>,
    val reports: List<DriverReport>,
)

data class QueueDetail(val directions: List<DirectionQueue>) {
    fun of(direction: Direction): DirectionQueue? = directions.firstOrNull { it.direction == direction }
}

/** A report to send; exactly the fields of its kind are used. */
data class NewReport(
    val direction: Direction,
    val kind: ReportKind,
    val vehiclesAhead: Int? = null,
    val waitMinutes: Int? = null,
    val note: String? = null,
)
