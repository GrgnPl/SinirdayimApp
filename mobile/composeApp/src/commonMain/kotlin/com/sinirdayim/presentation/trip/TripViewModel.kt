@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.presentation.trip

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.sinirdayim.domain.model.DriverState
import com.sinirdayim.domain.model.Place
import com.sinirdayim.domain.model.TripPlan
import com.sinirdayim.domain.model.TripRequest
import com.sinirdayim.domain.usecase.PlanTripUseCase
import com.sinirdayim.domain.usecase.SearchPlacesUseCase
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.LocalTime
import kotlinx.datetime.TimeZone
import kotlinx.datetime.atTime
import kotlinx.datetime.plus
import kotlinx.datetime.toInstant
import kotlinx.datetime.toLocalDateTime
import kotlin.time.Clock
import kotlin.time.Duration.Companion.hours
import kotlin.time.ExperimentalTime
import kotlin.time.Instant

enum class Endpoint { ORIGIN, DESTINATION }

enum class DepartOption(val label: String) {
    NOW("Şimdi"),
    IN_1H("1 sa sonra"),
    IN_3H("3 sa sonra"),
    TOMORROW_6("Yarın 06:00"),
}

data class PlaceField(
    val query: String = "",
    val selected: Place? = null,
    val suggestions: List<Place> = emptyList(),
)

data class TripUiState(
    val origin: PlaceField = PlaceField(),
    val destination: PlaceField = PlaceField(),
    val activeField: Endpoint? = null,
    val depart: DepartOption = DepartOption.NOW,
    val driver: DriverState = DriverState(),
    val isPlanning: Boolean = false,
    val plan: TripPlan? = null,
    val error: String? = null,
) {
    val canPlan: Boolean get() = origin.selected != null && destination.selected != null && !isPlanning
    fun field(e: Endpoint) = if (e == Endpoint.ORIGIN) origin else destination
}

class TripViewModel(
    private val searchPlaces: SearchPlacesUseCase,
    private val planTrip: PlanTripUseCase,
) : ViewModel() {
    private val _state = MutableStateFlow(TripUiState())
    val state: StateFlow<TripUiState> = _state.asStateFlow()

    private var searchJob: Job? = null

    fun onQueryChange(endpoint: Endpoint, query: String) {
        updateField(endpoint) { it.copy(query = query, selected = null) }
        _state.update { it.copy(activeField = endpoint) }
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(300) // debounce typing
            val results = try {
                searchPlaces(query)
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                emptyList()
            }
            updateField(endpoint) { it.copy(suggestions = results) }
        }
    }

    fun onSelect(endpoint: Endpoint, place: Place) {
        searchJob?.cancel()
        updateField(endpoint) { PlaceField(query = place.name, selected = place) }
        _state.update { it.copy(activeField = null) }
    }

    fun swap() = _state.update { it.copy(origin = it.destination, destination = it.origin) }

    fun onDepart(option: DepartOption) = _state.update { it.copy(depart = option) }

    fun onDriver(driver: DriverState) = _state.update { it.copy(driver = driver) }

    /** Plans the trip; calls [onDone] on success. */
    fun plan(onDone: () -> Unit) {
        val s = _state.value
        val from = s.origin.selected ?: return
        val to = s.destination.selected ?: return
        _state.update { it.copy(isPlanning = true, error = null, activeField = null) }
        viewModelScope.launch {
            try {
                val plan = planTrip(TripRequest(from.location, to.location, departAt(s.depart), s.driver))
                _state.update { it.copy(plan = plan, isPlanning = false) }
                onDone()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                _state.update { it.copy(isPlanning = false, error = "Rota planlanamadı. Noktaları kontrol edip tekrar dene.") }
            }
        }
    }

    private fun updateField(endpoint: Endpoint, f: (PlaceField) -> PlaceField) = _state.update {
        if (endpoint == Endpoint.ORIGIN) it.copy(origin = f(it.origin)) else it.copy(destination = f(it.destination))
    }

    private fun departAt(option: DepartOption): Instant {
        val now = Clock.System.now()
        return when (option) {
            DepartOption.NOW -> now
            DepartOption.IN_1H -> now + 1.hours
            DepartOption.IN_3H -> now + 3.hours
            DepartOption.TOMORROW_6 -> {
                val tz = TimeZone.currentSystemDefault()
                val tomorrow = now.toLocalDateTime(tz).date.plus(1, DateTimeUnit.DAY)
                tomorrow.atTime(LocalTime(6, 0)).toInstant(tz)
            }
        }
    }
}
