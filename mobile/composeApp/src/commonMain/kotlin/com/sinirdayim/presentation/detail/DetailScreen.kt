@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.presentation.detail

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.sinirdayim.domain.model.Confidence
import com.sinirdayim.domain.model.CrossingDetail
import com.sinirdayim.domain.model.Direction
import com.sinirdayim.domain.model.Snapshot
import com.sinirdayim.domain.model.WaitEstimate
import com.sinirdayim.presentation.components.DirectionToggle
import com.sinirdayim.presentation.components.LevelPill
import com.sinirdayim.presentation.components.StatTile
import com.sinirdayim.presentation.components.TrendChart
import com.sinirdayim.presentation.theme.LocalLevelColors
import com.sinirdayim.presentation.util.flagEmoji
import com.sinirdayim.presentation.util.formatDuration
import com.sinirdayim.presentation.util.formatDurationShort
import com.sinirdayim.presentation.util.formatKm
import com.sinirdayim.presentation.util.formatNumber
import com.sinirdayim.presentation.util.formatRelative
import kotlin.time.ExperimentalTime

@Composable
fun DetailScreen(viewModel: DetailViewModel, onBack: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = MaterialTheme.colorScheme

    Column(
        Modifier
            .fillMaxSize()
            .background(colors.background)
            .safeDrawingPadding()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 20.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Box(
            Modifier.size(40.dp).clip(CircleShape).background(colors.surface).clickable(onClick = onBack),
            contentAlignment = Alignment.Center,
        ) {
            Icon(Icons.AutoMirrored.Rounded.ArrowBack, contentDescription = "Geri", modifier = Modifier.size(20.dp))
        }

        val detail = state.detail
        when {
            detail != null -> Content(detail, state, viewModel)
            state.error != null -> Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
                Text(state.error!!, color = colors.onSurfaceVariant)
                TextButton(onClick = viewModel::refresh) { Text("Tekrar dene") }
            }
            else -> Text("Yükleniyor…", color = colors.onSurfaceVariant)
        }
    }
}

@Composable
private fun Content(detail: CrossingDetail, state: DetailUiState, viewModel: DetailViewModel) {
    val colors = MaterialTheme.colorScheme
    val direction = state.direction
    val onDirection = viewModel::selectDirection
    val crossing = detail.status.crossing
    val queue = state.queue?.of(direction)
    // The queue view includes driver reports; fall back to the list estimate.
    val est = queue?.estimate ?: detail.status.estimate(direction)
    val history = detail.history.filter { it.direction == direction }.sortedBy { it.observedAt }
    val latest = history.lastOrNull()
    val (a, b) = crossing.countries

    Column {
        Text(
            "${flagEmoji(a)} ${a}  ·  ${flagEmoji(b)} ${b}",
            style = MaterialTheme.typography.labelMedium,
            color = colors.onSurfaceVariant,
        )
        Spacer(Modifier.height(4.dp))
        Text(crossing.name, style = MaterialTheme.typography.headlineLarge, color = colors.onBackground)
    }

    DirectionToggle(
        selected = direction,
        onSelect = onDirection,
        labels = { if (it == Direction.EXPORT) "$a → $b" else "$b → $a" },
    )

    Hero(est)

    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        StatTile("Bekleyen araç", est.vehicles?.let { "~" + formatNumber(it) } ?: "—", Modifier.weight(1f), unit = "tır")
        StatTile("Kuyruk", latest?.queueKm?.let(::formatKm) ?: "—", Modifier.weight(1f), unit = "km")
    }
    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        StatTile("Günlük geçiş", latest?.dailyThroughput?.let(::formatNumber) ?: "—", Modifier.weight(1f), unit = "tır")
        StatTile("TIR parkı", latest?.parkedVehicles?.let(::formatNumber) ?: "—", Modifier.weight(1f), unit = "tır")
    }

    queue?.let {
        OutlookCard(it)
        SourcesCard(it)
        ReportsCard(it, state.sending, state.reportMessage, viewModel::report, viewModel::clearReportMessage)
    }

    TrendCard(history, LocalLevelColors.current.of(est.level))

    Text(
        methodNote(est),
        style = MaterialTheme.typography.labelMedium,
        color = colors.onSurfaceVariant,
    )
}

@Composable
private fun Hero(est: WaitEstimate) {
    val colors = MaterialTheme.colorScheme
    val levelColor = LocalLevelColors.current.of(est.level)
    Column(
        Modifier
            .fillMaxWidth()
            .clip(MaterialTheme.shapes.large)
            .background(colors.surface)
            .padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("Tahmini bekleme", style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant, modifier = Modifier.weight(1f))
            LevelPill(est.level)
        }
        if (est.waitMinutes != null) {
            val (value, unit) = formatDurationShort(est.waitMinutes)
            Row(verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(value, style = MaterialTheme.typography.displayLarge, color = levelColor)
                Text(unit, style = MaterialTheme.typography.headlineSmall, color = levelColor, modifier = Modifier.padding(bottom = 8.dp))
            }
            Text("≈ ${formatDuration(est.waitMinutes)}", style = MaterialTheme.typography.bodyMedium, color = colors.onSurfaceVariant)
        } else {
            Text("—", style = MaterialTheme.typography.displayLarge, color = colors.onSurfaceVariant)
            Text("Bu yön için geçiş hızı verisi yok", style = MaterialTheme.typography.bodyMedium, color = colors.onSurfaceVariant)
        }
        est.dataAt?.let {
            Text(
                "${est.sources.joinToString { if (it == "drivers") "Şoförler" else it.uppercase() }} · ${formatRelative(it)} güncellendi",
                style = MaterialTheme.typography.labelMedium,
                color = colors.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun TrendCard(history: List<Snapshot>, color: androidx.compose.ui.graphics.Color) {
    val colors = MaterialTheme.colorScheme
    val points = history.mapNotNull { s -> s.queueKm?.let { s.observedAt.epochSeconds.toDouble() to it } }
    Column(
        Modifier
            .fillMaxWidth()
            .clip(MaterialTheme.shapes.large)
            .background(colors.surface)
            .padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row {
            Text("Kuyruk uzunluğu", style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f))
            Text("son 3 gün · km", style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
        }
        if (points.isEmpty()) {
            Text("Henüz geçmiş veri yok", style = MaterialTheme.typography.bodyMedium, color = colors.onSurfaceVariant)
        } else {
            TrendChart(points, color = color, gridColor = colors.outlineVariant)
        }
    }
}

private fun methodNote(est: WaitEstimate): String {
    val method = when (est.method) {
        "reported" -> "Bekleme süresi kaynak tarafından bildirildi."
        "queue/throughput" -> "Tahmin: kuyruktaki araç sayısı ÷ saatlik geçiş hızı."
        "queue-only" -> "Yalnızca kuyruk uzunluğu biliniyor; süre hesaplanamadı."
        "driver_reports" -> "Bekleme süresi, son 3 saatte geçen şoförlerin bildirdiği sürelerin ortancası."
        "driver_queue/throughput" -> "Tahmin: şoförlerin bildirdiği kuyruk ÷ saatlik geçiş hızı."
        "driver_queue" -> "Kuyruk şoför bildirimlerinden; geçiş hızı bilinmiyor."
        else -> "Bu kapı için henüz veri yok."
    }
    val confidence = when (est.confidence) {
        Confidence.HIGH -> "Güven: yüksek"
        Confidence.MEDIUM -> "Güven: orta"
        Confidence.LOW -> "Güven: düşük"
    }
    return "$method $confidence."
}
