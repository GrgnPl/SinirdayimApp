package com.sinirbekleme.presentation.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.sinirbekleme.domain.model.CrossingStatus
import com.sinirbekleme.domain.model.Direction
import com.sinirbekleme.domain.usecase.GetCrossingsUseCase
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class HomeUiState(
    val direction: Direction = Direction.EXPORT,
    val query: String = "",
    val crossings: List<CrossingStatus> = emptyList(),
    val isLoading: Boolean = true,
    val error: String? = null,
) {
    val visible: List<CrossingStatus>
        get() = if (query.isBlank()) crossings
        else crossings.filter { it.crossing.name.contains(query.trim(), ignoreCase = true) }
}

class HomeViewModel(private val getCrossings: GetCrossingsUseCase) : ViewModel() {
    private val _state = MutableStateFlow(HomeUiState())
    val state: StateFlow<HomeUiState> = _state.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        _state.update { it.copy(isLoading = true, error = null) }
        viewModelScope.launch {
            try {
                val list = getCrossings(_state.value.direction)
                _state.update { it.copy(crossings = list, isLoading = false) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                _state.update { it.copy(isLoading = false, error = "Veriler alınamadı. Bağlantını kontrol edip tekrar dene.") }
            }
        }
    }

    fun selectDirection(direction: Direction) {
        if (direction == _state.value.direction) return
        _state.update { it.copy(direction = direction) }
        refresh()
    }

    fun search(query: String) = _state.update { it.copy(query = query) }
}
