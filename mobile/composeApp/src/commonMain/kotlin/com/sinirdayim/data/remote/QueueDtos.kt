package com.sinirdayim.data.remote

import kotlinx.serialization.Serializable

@Serializable
data class QueueDetailDto(val directions: List<DirectionQueueDto> = emptyList())

@Serializable
data class DirectionQueueDto(
    val direction: String,
    val estimate: WaitEstimateDto,
    val official: WaitEstimateDto,
    val dailyThroughput: Int? = null,
    val trend: TrendDto? = null,
    val outlook: List<HourOutlookDto> = emptyList(),
    val sources: List<SourceStatusDto> = emptyList(),
    val reports: List<DriverReportDto> = emptyList(),
)

@Serializable
data class TrendDto(val vehiclesPerHour: Double, val direction: String, val to: String)

@Serializable
data class HourOutlookDto(val at: String, val vehicles: Int, val waitMinutes: Int, val level: String)

@Serializable
data class SourceStatusDto(val source: String, val observedAt: String, val reports: Int = 0)

@Serializable
data class DriverReportDto(
    val kind: String,
    val at: String,
    val direction: String? = null,
    val vehiclesAhead: Int? = null,
    val queueKm: Double? = null,
    val waitMinutes: Int? = null,
    val note: String? = null,
)
