package com.sinirdayim.data.remote

import kotlinx.serialization.Serializable

@Serializable
data class PlaceListDto(val places: List<PlaceDto>)

@Serializable
data class PlaceDto(val name: String, val label: String, val country: String = "", val location: GeoPointDto)

@Serializable
data class PlanRequestDto(
    val origin: GeoPointDto,
    val destination: GeoPointDto,
    val departAt: String,
    val driver: DriverDto,
)

@Serializable
data class DriverDto(
    val continuousDrivingMin: Int,
    val dailyDrivingMin: Int,
    val extendedDaysLeft: Int,
)

@Serializable
data class TripPlanDto(
    val distanceKm: Double,
    val departure: String,
    val arrival: String,
    val totals: TripTotalsDto,
    val crossings: List<TripCrossingDto> = emptyList(),
    val steps: List<TripStepDto> = emptyList(),
    val polyline: String = "",
    val allWaitsKnown: Boolean = true,
    val alternatives: List<TripAlternativeDto> = emptyList(),
)

@Serializable
data class TripTotalsDto(
    val drivingMin: Int,
    val breakMin: Int,
    val dailyRestMin: Int,
    val borderWaitMin: Int,
    val totalMin: Int,
)

@Serializable
data class TripCrossingDto(
    val id: String,
    val name: String,
    val from: String,
    val to: String,
    val atKm: Double,
    val estimate: WaitEstimateDto,
)

@Serializable
data class TripStepDto(
    val kind: String,
    val start: String,
    val end: String,
    val durationMin: Int,
    val fromKm: Double,
    val toKm: Double,
    val location: GeoPointDto,
    val reason: String? = null,
    val crossingId: String? = null,
)

@Serializable
data class TripAlternativeDto(
    val via: List<String> = emptyList(),
    val distanceKm: Double,
    val totalMin: Int,
    val borderWaitMin: Int,
    val allWaitsKnown: Boolean,
    val selected: Boolean,
)
