package com.sinirbekleme.presentation.trip

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Place
import androidx.compose.material.icons.rounded.Remove
import androidx.compose.material.icons.rounded.SwapVert
import androidx.compose.material.icons.rounded.TripOrigin
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.sinirbekleme.domain.model.Place
import com.sinirbekleme.domain.usecase.PlanTripUseCase
import com.sinirbekleme.presentation.util.flagEmoji
import com.sinirbekleme.presentation.util.formatDuration

@Composable
fun TripFormScreen(viewModel: TripViewModel, onPlanned: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = MaterialTheme.colorScheme

    Column(
        Modifier
            .fillMaxSize()
            .background(colors.background)
            .safeDrawingPadding()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 20.dp, vertical = 24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Column(Modifier.padding(bottom = 8.dp)) {
            Text("Rota Planla", style = MaterialTheme.typography.headlineLarge, color = colors.onBackground)
            Text(
                "Takografa uygun molalar ve sınır beklemeleriyle",
                style = MaterialTheme.typography.bodyMedium,
                color = colors.onSurfaceVariant,
            )
        }

        EndpointsCard(state, viewModel)

        Section("Kalkış") {
            Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                DepartOption.entries.forEach { opt ->
                    Chip(opt.label, selected = opt == state.depart) { viewModel.onDepart(opt) }
                }
            }
        }

        Section("Takograf durumu") {
            Column(
                Modifier.fillMaxWidth().clip(MaterialTheme.shapes.large).background(colors.surface).padding(horizontal = 16.dp),
            ) {
                val d = state.driver
                StepperRow(
                    label = "Son moladan beri sürüş",
                    value = formatDuration(d.continuousDrivingMin),
                    onMinus = { viewModel.onDriver(d.copy(continuousDrivingMin = (d.continuousDrivingMin - 15).coerceAtLeast(0))) },
                    onPlus = {
                        val next = (d.continuousDrivingMin + 15).coerceAtMost(PlanTripUseCase.MAX_CONTINUOUS_MIN)
                        viewModel.onDriver(d.copy(continuousDrivingMin = next, dailyDrivingMin = maxOf(d.dailyDrivingMin, next)))
                    },
                )
                HorizontalDivider(color = colors.outlineVariant)
                StepperRow(
                    label = "Bugünkü toplam sürüş",
                    value = formatDuration(d.dailyDrivingMin),
                    onMinus = {
                        val next = (d.dailyDrivingMin - 15).coerceAtLeast(0)
                        viewModel.onDriver(d.copy(dailyDrivingMin = next, continuousDrivingMin = minOf(d.continuousDrivingMin, next)))
                    },
                    onPlus = { viewModel.onDriver(d.copy(dailyDrivingMin = (d.dailyDrivingMin + 15).coerceAtMost(PlanTripUseCase.MAX_DAILY_MIN))) },
                )
                HorizontalDivider(color = colors.outlineVariant)
                StepperRow(
                    label = "Bu hafta kalan 10 saatlik gün",
                    value = d.extendedDaysLeft.toString(),
                    onMinus = { viewModel.onDriver(d.copy(extendedDaysLeft = (d.extendedDaysLeft - 1).coerceAtLeast(0))) },
                    onPlus = { viewModel.onDriver(d.copy(extendedDaysLeft = (d.extendedDaysLeft + 1).coerceAtMost(2))) },
                )
            }
        }

        state.error?.let { Text(it, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.error) }

        Button(
            onClick = { viewModel.plan(onPlanned) },
            enabled = state.canPlan,
            shape = RoundedCornerShape(14.dp),
            colors = ButtonDefaults.buttonColors(containerColor = colors.primary, contentColor = colors.onPrimary),
            modifier = Modifier.fillMaxWidth().height(52.dp),
        ) {
            if (state.isPlanning) {
                CircularProgressIndicator(Modifier.size(20.dp), color = colors.onPrimary, strokeWidth = 2.dp)
                Spacer(Modifier.width(10.dp))
                Text("Rotalar hesaplanıyor…")
            } else {
                Text("Planla", style = MaterialTheme.typography.titleMedium)
            }
        }
    }
}

@Composable
private fun EndpointsCard(state: TripUiState, viewModel: TripViewModel) {
    val colors = MaterialTheme.colorScheme
    Column(Modifier.fillMaxWidth().clip(MaterialTheme.shapes.large).background(colors.surface)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                EndpointField(Icons.Rounded.TripOrigin, "Nereden", state.origin.query) { viewModel.onQueryChange(Endpoint.ORIGIN, it) }
                HorizontalDivider(Modifier.padding(start = 48.dp), color = colors.outlineVariant)
                EndpointField(Icons.Rounded.Place, "Nereye", state.destination.query) { viewModel.onQueryChange(Endpoint.DESTINATION, it) }
            }
            Box(
                Modifier.padding(end = 8.dp).size(40.dp).clip(CircleShape).clickable(onClick = viewModel::swap),
                contentAlignment = Alignment.Center,
            ) {
                Icon(Icons.Rounded.SwapVert, contentDescription = "Yer değiştir", tint = colors.onSurfaceVariant)
            }
        }
        state.activeField?.let { active ->
            val suggestions = state.field(active).suggestions
            if (suggestions.isNotEmpty()) {
                HorizontalDivider(color = colors.outlineVariant)
                suggestions.forEach { place -> Suggestion(place) { viewModel.onSelect(active, place) } }
            }
        }
    }
}

@Composable
private fun EndpointField(icon: ImageVector, hint: String, value: String, onChange: (String) -> Unit) {
    val colors = MaterialTheme.colorScheme
    Row(Modifier.fillMaxWidth().height(52.dp).padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically) {
        Icon(icon, contentDescription = null, tint = colors.onSurfaceVariant, modifier = Modifier.size(18.dp))
        Spacer(Modifier.width(14.dp))
        Box(Modifier.weight(1f)) {
            if (value.isEmpty()) Text(hint, style = MaterialTheme.typography.bodyMedium, color = colors.onSurfaceVariant)
            BasicTextField(
                value = value,
                onValueChange = onChange,
                singleLine = true,
                textStyle = MaterialTheme.typography.bodyMedium.copy(color = colors.onSurface),
                cursorBrush = SolidColor(colors.onSurface),
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }
}

@Composable
private fun Suggestion(place: Place, onClick: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onClick).padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(flagEmoji(place.country), modifier = Modifier.width(32.dp))
        Column(Modifier.weight(1f)) {
            Text(place.name, style = MaterialTheme.typography.bodyMedium, color = colors.onSurface, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text(place.label, style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

@Composable
private fun Section(title: String, content: @Composable () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(title, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        content()
    }
}

@Composable
private fun Chip(label: String, selected: Boolean, onClick: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    Box(
        Modifier
            .clip(CircleShape)
            .background(if (selected) colors.primary else colors.surface)
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 8.dp),
    ) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = if (selected) colors.onPrimary else colors.onSurface)
    }
}

@Composable
private fun StepperRow(label: String, value: String, onMinus: () -> Unit, onPlus: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    Row(Modifier.fillMaxWidth().height(56.dp), verticalAlignment = Alignment.CenterVertically) {
        Text(label, style = MaterialTheme.typography.bodyMedium, color = colors.onSurface, modifier = Modifier.weight(1f))
        RoundIcon(Icons.Rounded.Remove, "Azalt", onMinus)
        Text(value, style = MaterialTheme.typography.titleMedium, modifier = Modifier.width(84.dp), textAlign = androidx.compose.ui.text.style.TextAlign.Center)
        RoundIcon(Icons.Rounded.Add, "Artır", onPlus)
    }
}

@Composable
private fun RoundIcon(icon: ImageVector, description: String, onClick: () -> Unit) {
    Box(
        Modifier.size(32.dp).clip(CircleShape).background(MaterialTheme.colorScheme.surfaceVariant).clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(icon, contentDescription = description, modifier = Modifier.size(16.dp))
    }
}
