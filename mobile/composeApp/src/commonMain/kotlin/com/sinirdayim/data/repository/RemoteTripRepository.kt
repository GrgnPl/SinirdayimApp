package com.sinirdayim.data.repository

import com.sinirdayim.data.remote.TripApi
import com.sinirdayim.data.remote.toDomain
import com.sinirdayim.data.remote.toDto
import com.sinirdayim.domain.model.Place
import com.sinirdayim.domain.model.TripPlan
import com.sinirdayim.domain.model.TripRequest
import com.sinirdayim.domain.repository.TripRepository

class RemoteTripRepository(private val api: TripApi) : TripRepository {
    override suspend fun searchPlaces(query: String): List<Place> =
        api.places(query).places.map { it.toDomain() }

    override suspend fun plan(request: TripRequest): TripPlan =
        api.plan(request.toDto()).toDomain()
}
