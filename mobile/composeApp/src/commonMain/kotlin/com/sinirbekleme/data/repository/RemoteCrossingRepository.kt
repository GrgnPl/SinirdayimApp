package com.sinirbekleme.data.repository

import com.sinirbekleme.data.remote.CrossingApi
import com.sinirbekleme.data.remote.toDomain
import com.sinirbekleme.domain.model.CrossingDetail
import com.sinirbekleme.domain.model.CrossingStatus
import com.sinirbekleme.domain.repository.CrossingRepository

class RemoteCrossingRepository(private val api: CrossingApi) : CrossingRepository {
    override suspend fun getCrossings(): List<CrossingStatus> =
        api.crossings().crossings.map { it.toDomain() }

    override suspend fun getCrossing(id: String, historyHours: Int): CrossingDetail =
        api.crossing(id, historyHours).toDomain()
}
