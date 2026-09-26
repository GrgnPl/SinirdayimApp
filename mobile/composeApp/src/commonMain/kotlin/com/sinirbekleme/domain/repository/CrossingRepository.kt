package com.sinirbekleme.domain.repository

import com.sinirbekleme.domain.model.CrossingDetail
import com.sinirbekleme.domain.model.CrossingStatus

interface CrossingRepository {
    suspend fun getCrossings(): List<CrossingStatus>
    suspend fun getCrossing(id: String, historyHours: Int): CrossingDetail
}
