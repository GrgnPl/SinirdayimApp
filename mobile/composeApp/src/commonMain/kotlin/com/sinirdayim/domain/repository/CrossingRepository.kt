package com.sinirdayim.domain.repository

import com.sinirdayim.domain.model.CrossingDetail
import com.sinirdayim.domain.model.CrossingStatus

interface CrossingRepository {
    suspend fun getCrossings(): List<CrossingStatus>
    suspend fun getCrossing(id: String, historyHours: Int): CrossingDetail
}
