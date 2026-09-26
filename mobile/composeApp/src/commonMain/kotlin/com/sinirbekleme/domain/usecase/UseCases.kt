package com.sinirbekleme.domain.usecase

import com.sinirbekleme.domain.model.CrossingDetail
import com.sinirbekleme.domain.model.CrossingStatus
import com.sinirbekleme.domain.model.Direction
import com.sinirbekleme.domain.model.Level
import com.sinirbekleme.domain.repository.CrossingRepository

/** Crossings for a direction: those with data first, longest wait first. */
class GetCrossingsUseCase(private val repository: CrossingRepository) {
    suspend operator fun invoke(direction: Direction): List<CrossingStatus> =
        repository.getCrossings().sortedWith(
            compareBy<CrossingStatus> { it.estimate(direction).level == Level.UNKNOWN }
                .thenByDescending { it.estimate(direction).waitMinutes ?: -1 }
                .thenByDescending { it.estimate(direction).vehicles ?: -1 }
                .thenBy { it.crossing.name },
        )
}

class GetCrossingDetailUseCase(private val repository: CrossingRepository) {
    suspend operator fun invoke(id: String, historyHours: Int = 72): CrossingDetail =
        repository.getCrossing(id, historyHours)
}
