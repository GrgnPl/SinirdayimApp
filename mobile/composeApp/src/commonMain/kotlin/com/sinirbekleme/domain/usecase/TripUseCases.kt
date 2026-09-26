package com.sinirbekleme.domain.usecase

import com.sinirbekleme.domain.model.Place
import com.sinirbekleme.domain.model.TripPlan
import com.sinirbekleme.domain.model.TripRequest
import com.sinirbekleme.domain.repository.TripRepository

class SearchPlacesUseCase(private val repository: TripRepository) {
    suspend operator fun invoke(query: String): List<Place> =
        if (query.trim().length < 2) emptyList() else repository.searchPlaces(query.trim())
}

class PlanTripUseCase(private val repository: TripRepository) {
    suspend operator fun invoke(request: TripRequest): TripPlan {
        require(request.driver.continuousDrivingMin in 0..MAX_CONTINUOUS_MIN) { "continuous driving out of range" }
        require(request.driver.dailyDrivingMin in 0..MAX_DAILY_MIN) { "daily driving out of range" }
        return repository.plan(request)
    }

    companion object {
        const val MAX_CONTINUOUS_MIN = 270 // 4h30
        const val MAX_DAILY_MIN = 600 // 10h
    }
}
