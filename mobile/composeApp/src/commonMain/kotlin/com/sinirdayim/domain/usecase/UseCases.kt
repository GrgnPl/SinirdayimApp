package com.sinirdayim.domain.usecase

import com.sinirdayim.domain.model.CrossingDetail
import com.sinirdayim.domain.model.CrossingStatus
import com.sinirdayim.domain.model.Direction
import com.sinirdayim.domain.model.Level
import com.sinirdayim.domain.model.NewReport
import com.sinirdayim.domain.model.QueueDetail
import com.sinirdayim.domain.model.ReportKind
import com.sinirdayim.domain.repository.CrossingRepository

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

class GetQueueUseCase(private val repository: CrossingRepository) {
    suspend operator fun invoke(id: String): QueueDetail = repository.getQueue(id)
}

class SendReportUseCase(private val repository: CrossingRepository) {
    suspend operator fun invoke(crossingId: String, report: NewReport) {
        when (report.kind) {
            ReportKind.IN_QUEUE -> require(report.vehiclesAhead != null && report.vehiclesAhead >= 0) { "vehicles ahead required" }
            ReportKind.PASSED -> require(report.waitMinutes != null && report.waitMinutes >= 0) { "wait required" }
        }
        repository.report(crossingId, report.copy(note = report.note?.trim()?.take(280)?.ifBlank { null }))
    }
}

class GetCrossingDetailUseCase(private val repository: CrossingRepository) {
    suspend operator fun invoke(id: String, historyHours: Int = 72): CrossingDetail =
        repository.getCrossing(id, historyHours)
}
