@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.data.remote

import com.sinirdayim.domain.model.Direction
import com.sinirdayim.domain.model.DirectionQueue
import com.sinirdayim.domain.model.DriverReport
import com.sinirdayim.domain.model.HourOutlook
import com.sinirdayim.domain.model.Level
import com.sinirdayim.domain.model.NewReport
import com.sinirdayim.domain.model.QueueDetail
import com.sinirdayim.domain.model.QueueTrend
import com.sinirdayim.domain.model.ReportKind
import com.sinirdayim.domain.model.SourceStatus
import com.sinirdayim.domain.model.TrendDirection
import kotlin.time.ExperimentalTime
import kotlin.time.Instant

fun QueueDetailDto.toDomain() = QueueDetail(directions.mapNotNull { it.toDomainOrNull() })

private fun DirectionQueueDto.toDomainOrNull(): DirectionQueue? {
    val dir = when (direction) {
        "export" -> Direction.EXPORT
        "import" -> Direction.IMPORT
        else -> return null
    }
    return DirectionQueue(
        direction = dir,
        estimate = estimate.toDomain(),
        official = official.toDomain(),
        dailyThroughput = dailyThroughput,
        trend = trend?.let { t ->
            instant(t.to)?.let { to ->
                QueueTrend(
                    t.vehiclesPerHour,
                    when (t.direction) {
                        "growing" -> TrendDirection.GROWING
                        "shrinking" -> TrendDirection.SHRINKING
                        else -> TrendDirection.STABLE
                    },
                    to,
                )
            }
        },
        outlook = outlook.mapNotNull { o ->
            instant(o.at)?.let { HourOutlook(it, o.vehicles, o.waitMinutes, level(o.level)) }
        },
        sources = sources.mapNotNull { s -> instant(s.observedAt)?.let { SourceStatus(s.source, it, s.reports) } },
        reports = reports.mapNotNull { r ->
            val at = instant(r.at) ?: return@mapNotNull null
            val kind = if (r.kind == "passed") ReportKind.PASSED else ReportKind.IN_QUEUE
            DriverReport(kind, at, r.vehiclesAhead, r.queueKm, r.waitMinutes, r.note?.takeIf { it.isNotBlank() })
        },
    )
}

fun NewReport.toDto() = DriverReportDto(
    kind = if (kind == ReportKind.PASSED) "passed" else "in_queue",
    at = kotlin.time.Clock.System.now().toString(),
    direction = if (direction == Direction.EXPORT) "export" else "import",
    vehiclesAhead = vehiclesAhead,
    waitMinutes = waitMinutes,
    note = note,
)

private fun level(v: String) = when (v) {
    "low" -> Level.LOW
    "medium" -> Level.MEDIUM
    "high" -> Level.HIGH
    else -> Level.UNKNOWN
}

private fun instant(v: String): Instant? = runCatching { Instant.parse(v) }.getOrNull()
