package com.sinirdayim.domain.repository

import com.sinirdayim.domain.model.Place
import com.sinirdayim.domain.model.TripPlan
import com.sinirdayim.domain.model.TripRequest

interface TripRepository {
    suspend fun searchPlaces(query: String): List<Place>
    suspend fun plan(request: TripRequest): TripPlan
}
