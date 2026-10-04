package com.sinirdayim.di

import com.sinirdayim.data.remote.CrossingApi
import com.sinirdayim.data.remote.TripApi
import com.sinirdayim.data.repository.RemoteCrossingRepository
import com.sinirdayim.data.repository.RemoteTripRepository
import com.sinirdayim.domain.repository.CrossingRepository
import com.sinirdayim.domain.repository.TripRepository
import com.sinirdayim.domain.usecase.GetCrossingDetailUseCase
import com.sinirdayim.domain.usecase.GetCrossingsUseCase
import com.sinirdayim.domain.usecase.GetQueueUseCase
import com.sinirdayim.domain.usecase.SendReportUseCase
import com.sinirdayim.domain.usecase.PlanTripUseCase
import com.sinirdayim.domain.usecase.SearchPlacesUseCase

/** Manual DI: the single place where implementations are chosen. */
class AppContainer(apiBaseUrl: String) {
    private val httpClient = CrossingApi.createClient(apiBaseUrl)

    private val crossingRepository: CrossingRepository = RemoteCrossingRepository(CrossingApi(httpClient))
    private val tripRepository: TripRepository = RemoteTripRepository(TripApi(httpClient))

    val getCrossings = GetCrossingsUseCase(crossingRepository)
    val getCrossingDetail = GetCrossingDetailUseCase(crossingRepository)
    val getQueue = GetQueueUseCase(crossingRepository)
    val sendReport = SendReportUseCase(crossingRepository)
    val searchPlaces = SearchPlacesUseCase(tripRepository)
    val planTrip = PlanTripUseCase(tripRepository)
}
