package com.sinirbekleme.data.repository

import com.sinirbekleme.data.remote.TripApi
import com.sinirbekleme.data.remote.toDomain
import com.sinirbekleme.data.remote.toDto
import com.sinirbekleme.domain.model.Place
import com.sinirbekleme.domain.model.TripPlan
import com.sinirbekleme.domain.model.TripRequest
import com.sinirbekleme.domain.repository.TripRepository

class RemoteTripRepository(private val api: TripApi) : TripRepository {
    override suspend fun searchPlaces(query: String): List<Place> =
        api.places(query).places.map { it.toDomain() }

    override suspend fun plan(request: TripRequest): TripPlan =
        api.plan(request.toDto()).toDomain()
}
