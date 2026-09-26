package com.sinirbekleme.di

import com.sinirbekleme.data.remote.CrossingApi
import com.sinirbekleme.data.repository.RemoteCrossingRepository
import com.sinirbekleme.domain.repository.CrossingRepository
import com.sinirbekleme.domain.usecase.GetCrossingDetailUseCase
import com.sinirbekleme.domain.usecase.GetCrossingsUseCase

/** Manual DI: the single place where implementations are chosen. */
class AppContainer(apiBaseUrl: String) {
    private val repository: CrossingRepository =
        RemoteCrossingRepository(CrossingApi(CrossingApi.createClient(apiBaseUrl)))

    val getCrossings = GetCrossingsUseCase(repository)
    val getCrossingDetail = GetCrossingDetailUseCase(repository)
}
