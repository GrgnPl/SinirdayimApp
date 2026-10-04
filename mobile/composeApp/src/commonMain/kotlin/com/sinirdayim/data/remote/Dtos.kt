package com.sinirdayim.data.remote

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class CrossingListDto(val crossings: List<CrossingStatusDto>)

@Serializable
data class CrossingStatusDto(
    val id: String,
    val name: String,
    val countries: List<String>,
    val location: GeoPointDto,
    val export: WaitEstimateDto,
    @SerialName("import") val importEstimate: WaitEstimateDto,
)

@Serializable
data class CrossingDetailDto(
    val id: String,
    val name: String,
    val countries: List<String>,
    val location: GeoPointDto,
    val export: WaitEstimateDto,
    @SerialName("import") val importEstimate: WaitEstimateDto,
    val history: List<SnapshotDto> = emptyList(),
)

@Serializable
data class GeoPointDto(val lat: Double, val lng: Double)

@Serializable
data class WaitEstimateDto(
    val vehicles: Int? = null,
    val waitMinutes: Int? = null,
    val level: String,
    val confidence: String,
    val method: String,
    val basedOn: List<String> = emptyList(),
    val dataAt: String? = null,
)

@Serializable
data class SnapshotDto(
    val direction: String,
    val source: String,
    val observedAt: String,
    val queueKm: Double? = null,
    val queueVehicles: Int? = null,
    val parkedVehicles: Int? = null,
    val dailyThroughput: Int? = null,
    val waitMinutes: Int? = null,
)
