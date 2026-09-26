package com.sinirbekleme.di

import com.sinirbekleme.data.remote.CrossingApi
import com.sinirbekleme.data.remote.TripApi
import com.sinirbekleme.data.repository.RemoteCrossingRepository
import com.sinirbekleme.data.repository.RemoteTripRepository
import com.sinirbekleme.domain.repository.CrossingRepository
import com.sinirbekleme.domain.repository.TripRepository
import com.sinirbekleme.domain.usecase.GetCrossingDetailUseCase
import com.sinirbekleme.domain.usecase.GetCrossingsUseCase
import com.sinirbekleme.domain.usecase.PlanTripUseCase
import com.sinirbekleme.domain.usecase.SearchPlacesUseCase

/** Manual DI: the single place where implementations are chosen. */
class AppContainer(apiBaseUrl: String) {
    private val httpClient = CrossingApi.createClient(apiBaseUrl)

    private val crossingRepository: CrossingRepository = RemoteCrossingRepository(CrossingApi(httpClient))
    private val tripRepository: TripRepository = RemoteTripRepository(TripApi(httpClient))

    val getCrossings = GetCrossingsUseCase(crossingRepository)
    val getCrossingDetail = GetCrossingDetailUseCase(crossingRepository)
    val searchPlaces = SearchPlacesUseCase(tripRepository)
    val planTrip = PlanTripUseCase(tripRepository)
}
