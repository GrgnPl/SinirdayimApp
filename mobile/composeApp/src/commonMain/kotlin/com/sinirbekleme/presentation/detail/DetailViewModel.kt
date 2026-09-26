package com.sinirbekleme.presentation.detail

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.sinirbekleme.domain.model.CrossingDetail
import com.sinirbekleme.domain.model.Direction
import com.sinirbekleme.domain.usecase.GetCrossingDetailUseCase
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class DetailUiState(
    val direction: Direction,
    val detail: CrossingDetail? = null,
    val isLoading: Boolean = true,
    val error: String? = null,
)

class DetailViewModel(
    private val crossingId: String,
    initialDirection: Direction,
    private val getDetail: GetCrossingDetailUseCase,
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
                val detail = getDetail(crossingId)
                _state.update { it.copy(detail = detail, isLoading = false) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                _state.update { it.copy(isLoading = false, error = "Veriler alınamadı.") }
            }
        }
    }

    fun selectDirection(direction: Direction) = _state.update { it.copy(direction = direction) }
}
