package com.sinirbekleme.presentation.home

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.sinirbekleme.domain.model.CrossingStatus
import com.sinirbekleme.domain.model.Direction
import com.sinirbekleme.presentation.components.DirectionToggle
import com.sinirbekleme.presentation.components.LevelDot
import com.sinirbekleme.presentation.theme.LocalLevelColors
import com.sinirbekleme.presentation.util.flagEmoji
import com.sinirbekleme.presentation.util.formatDurationShort
import com.sinirbekleme.presentation.util.formatNumber
import com.sinirbekleme.presentation.util.formatRelative
import com.sinirbekleme.presentation.util.label

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    viewModel: HomeViewModel,
    onOpen: (id: String, direction: Direction) -> Unit,
) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).safeDrawingPadding()) {
        PullToRefreshBox(
            isRefreshing = state.isLoading && state.crossings.isNotEmpty(),
            onRefresh = viewModel::refresh,
        ) {
            LazyColumn(
                contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = 24.dp, bottom = 32.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
                modifier = Modifier.fillMaxSize(),
            ) {
                item { Header() }
                item {
                    DirectionToggle(
                        selected = state.direction,
                        onSelect = viewModel::selectDirection,
                        labels = { if (it == Direction.EXPORT) "Türkiye'den çıkış" else "Türkiye'ye giriş" },
                    )
                }
                item { SearchField(state.query, viewModel::search) }

                when {
                    state.error != null && state.crossings.isEmpty() -> item {
                        Message(state.error!!, action = "Tekrar dene", onAction = viewModel::refresh)
                    }
                    state.isLoading && state.crossings.isEmpty() -> item { Message("Yükleniyor…") }
                    state.visible.isEmpty() -> item { Message("Sonuç bulunamadı") }
                    else -> items(state.visible, key = { it.crossing.id }) { item ->
                        CrossingCard(item, state.direction, onClick = { onOpen(item.crossing.id, state.direction) })
                    }
                }
            }
        }
    }
}

@Composable
private fun Header() {
    Column(Modifier.padding(bottom = 8.dp)) {
        Text("Sınır Kapıları", style = MaterialTheme.typography.headlineLarge, color = MaterialTheme.colorScheme.onBackground)
        Text(
            "Tır yoğunluğu ve tahmini bekleme",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun SearchField(query: String, onChange: (String) -> Unit) {
    val colors = MaterialTheme.colorScheme
    Row(
        Modifier
            .fillMaxWidth()
            .height(44.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(colors.surface)
            .border(1.dp, colors.outlineVariant, RoundedCornerShape(12.dp))
            .padding(horizontal = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(Icons.Rounded.Search, contentDescription = null, tint = colors.onSurfaceVariant, modifier = Modifier.size(18.dp))
        Spacer(Modifier.width(8.dp))
        Box(Modifier.weight(1f)) {
            if (query.isEmpty()) {
                Text("Kapı ara", style = MaterialTheme.typography.bodyMedium, color = colors.onSurfaceVariant)
            }
            BasicTextField(
                value = query,
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
private fun CrossingCard(item: CrossingStatus, direction: Direction, onClick: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    val est = item.estimate(direction)
    val levelColor = LocalLevelColors.current.of(est.level)
    val (from, to) = item.crossing.countries.let { if (direction == Direction.EXPORT) it else it.second to it.first }

    Column(
        Modifier
            .fillMaxWidth()
            .clip(MaterialTheme.shapes.large)
            .background(colors.surface)
            .clickable(onClick = onClick)
            .padding(18.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(
                    "${flagEmoji(from)}  →  ${flagEmoji(to)}",
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.onSurfaceVariant,
                )
                Spacer(Modifier.height(4.dp))
                Text(
                    item.crossing.name,
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.onSurface,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            WaitValue(est.waitMinutes, levelColor)
        }

        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            LevelDot(est.level)
            Text(est.level.label(), style = MaterialTheme.typography.labelMedium, color = levelColor)
            est.vehicles?.let {
                Text("·", color = colors.onSurfaceVariant)
                Text("~${formatNumber(it)} tır", style = MaterialTheme.typography.labelMedium, color = colors.onSurface)
            }
            Spacer(Modifier.weight(1f))
            est.dataAt?.let {
                Text(formatRelative(it), style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
            }
        }
    }
}

@Composable
private fun WaitValue(minutes: Int?, color: androidx.compose.ui.graphics.Color) {
    if (minutes == null) {
        Text("—", style = MaterialTheme.typography.headlineSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        return
    }
    val (value, unit) = formatDurationShort(minutes)
    Row(verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(3.dp)) {
        Text(value, style = MaterialTheme.typography.headlineSmall, color = color)
        Text(unit, style = MaterialTheme.typography.labelMedium, color = color, modifier = Modifier.padding(bottom = 3.dp))
    }
}

@Composable
private fun Message(text: String, action: String? = null, onAction: () -> Unit = {}) {
    Column(
        Modifier.fillMaxWidth().padding(vertical = 48.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(text, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        if (action != null) TextButton(onClick = onAction) { Text(action) }
    }
}
