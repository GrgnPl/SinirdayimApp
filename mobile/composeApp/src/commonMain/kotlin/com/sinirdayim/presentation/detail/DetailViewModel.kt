package com.sinirdayim.presentation.detail

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.sinirdayim.domain.model.CrossingDetail
import com.sinirdayim.domain.model.Direction
import com.sinirdayim.domain.model.NewReport
import com.sinirdayim.domain.model.QueueDetail
import com.sinirdayim.domain.usecase.GetCrossingDetailUseCase
import com.sinirdayim.domain.usecase.GetQueueUseCase
import com.sinirdayim.domain.usecase.SendReportUseCase
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.async
import kotlinx.coroutines.launch

data class DetailUiState(
    val direction: Direction,
    val detail: CrossingDetail? = null,
    val queue: QueueDetail? = null,
    val isLoading: Boolean = true,
    val error: String? = null,
    val sending: Boolean = false,
    /** Result of the last report: a thank-you or an error. */
    val reportMessage: String? = null,
)

class DetailViewModel(
    private val crossingId: String,
    initialDirection: Direction,
    private val getDetail: GetCrossingDetailUseCase,
    private val getQueue: GetQueueUseCase,
    private val sendReport: SendReportUseCase,
) : ViewModel() {
    private val _state = MutableStateFlow(DetailUiState(direction = initialDirection))
    val state: StateFlow<DetailUiState> = _state.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        _state.update { it.copy(isLoading = true, error = null) }
        viewModelScope.launch {
            try {
                val detail = async { getDetail(crossingId) }
                // The queue view is extra detail; the screen works without it.
                val queue = async { runCatching { getQueue(crossingId) }.getOrNull() }
                _state.update { it.copy(detail = detail.await(), queue = queue.await(), isLoading = false) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                _state.update { it.copy(isLoading = false, error = "Veriler alınamadı.") }
            }
        }
    }

    fun selectDirection(direction: Direction) = _state.update { it.copy(direction = direction) }

    fun report(report: NewReport) {
        _state.update { it.copy(sending = true, reportMessage = null) }
        viewModelScope.launch {
            try {
                sendReport(crossingId, report)
                val queue = runCatching { getQueue(crossingId) }.getOrNull()
                _state.update { it.copy(sending = false, queue = queue ?: it.queue, reportMessage = "Teşekkürler, bildirimin eklendi.") }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                _state.update { it.copy(sending = false, reportMessage = "Bildirim gönderilemedi.") }
            }
        }
    }

    fun clearReportMessage() = _state.update { it.copy(reportMessage = null) }
}
