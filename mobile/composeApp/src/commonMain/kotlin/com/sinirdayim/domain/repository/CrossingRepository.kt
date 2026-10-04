package com.sinirdayim.domain.repository

import com.sinirdayim.domain.model.CrossingDetail
import com.sinirdayim.domain.model.CrossingStatus
import com.sinirdayim.domain.model.NewReport
import com.sinirdayim.domain.model.QueueDetail

interface CrossingRepository {
    suspend fun getCrossings(): List<CrossingStatus>
    suspend fun getCrossing(id: String, historyHours: Int): CrossingDetail
    suspend fun getQueue(id: String): QueueDetail
    suspend fun report(id: String, report: NewReport)
}
