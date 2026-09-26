package com.sinirbekleme.domain.repository

import com.sinirbekleme.domain.model.Place
import com.sinirbekleme.domain.model.TripPlan
import com.sinirbekleme.domain.model.TripRequest

interface TripRepository {
    suspend fun searchPlaces(query: String): List<Place>
    suspend fun plan(request: TripRequest): TripPlan
}
