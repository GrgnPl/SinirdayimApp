package com.sinirdayim.data.repository

import com.sinirdayim.data.remote.CrossingApi
import com.sinirdayim.data.remote.toDomain
import com.sinirdayim.data.remote.toDto
import com.sinirdayim.domain.model.CrossingDetail
import com.sinirdayim.domain.model.CrossingStatus
import com.sinirdayim.domain.model.NewReport
import com.sinirdayim.domain.model.QueueDetail
import com.sinirdayim.domain.repository.CrossingRepository

class RemoteCrossingRepository(private val api: CrossingApi) : CrossingRepository {
    override suspend fun getCrossings(): List<CrossingStatus> =
        api.crossings().crossings.map { it.toDomain() }

    override suspend fun getCrossing(id: String, historyHours: Int): CrossingDetail =
        api.crossing(id, historyHours).toDomain()

    override suspend fun getQueue(id: String): QueueDetail = api.queue(id).toDomain()

    override suspend fun report(id: String, report: NewReport) = api.report(id, report.toDto())
}
