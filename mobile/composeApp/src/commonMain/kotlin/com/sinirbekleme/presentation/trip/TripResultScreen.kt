package com.sinirbekleme.presentation.trip

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.Flag
import androidx.compose.material.icons.rounded.Hotel
import androidx.compose.material.icons.rounded.LocalCafe
import androidx.compose.material.icons.rounded.LocalShipping
import androidx.compose.material.icons.rounded.Place
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.sinirbekleme.domain.model.StepKind
import com.sinirbekleme.domain.model.StopReason
import com.sinirbekleme.domain.model.TripAlternative
import com.sinirbekleme.domain.model.TripPlan
import com.sinirbekleme.domain.model.TripStep
import com.sinirbekleme.presentation.components.StatTile
import com.sinirbekleme.presentation.theme.LocalLevelColors
import com.sinirbekleme.presentation.util.formatClock
import com.sinirbekleme.presentation.util.formatDuration
import com.sinirbekleme.presentation.util.formatNumber
import kotlin.math.roundToInt

@Composable
fun TripResultScreen(viewModel: TripViewModel, onBack: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val plan = state.plan ?: return
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

        Text(
            "${state.origin.selected?.name.orEmpty()} → ${state.destination.selected?.name.orEmpty()}",
            style = MaterialTheme.typography.headlineSmall,
            color = colors.onBackground,
        )

        ArrivalHero(plan)

        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            StatTile("Sürüş", formatDuration(plan.totals.drivingMin), Modifier.weight(1f))
            StatTile("Sınır", formatDuration(plan.totals.borderWaitMin), Modifier.weight(1f))
        }
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            StatTile("Mola", formatDuration(plan.totals.breakMin), Modifier.weight(1f))
            StatTile("Günlük dinlenme", formatDuration(plan.totals.dailyRestMin), Modifier.weight(1f))
        }

        if (plan.alternatives.size > 1) Alternatives(plan.alternatives)

        Timeline(plan)

        Text(
            "Plan AB 561/2006 ve AETR kurallarına göre hesaplandı: 4,5 saatte 45 dk mola, günde en fazla 9 saat " +
                "(haftada 2 gün 10 saat) sürüş, 11 saat günlük dinlenme. Sınırda 45 dk üzeri bekleme mola, " +
                "11 saat üzeri bekleme günlük dinlenme sayılır.",
            style = MaterialTheme.typography.labelMedium,
            color = colors.onSurfaceVariant,
        )
    }
}

@Composable
private fun ArrivalHero(plan: TripPlan) {
    val colors = MaterialTheme.colorScheme
    Column(
        Modifier.fillMaxWidth().clip(MaterialTheme.shapes.large).background(colors.surface).padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Text("Tahmini varış", style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
        Text(formatClock(plan.arrival), style = MaterialTheme.typography.displayLarge, color = colors.onSurface)
        Text(
            "${formatDuration(plan.totals.totalMin)} · ${formatNumber(plan.distanceKm.roundToInt())} km",
            style = MaterialTheme.typography.bodyMedium,
            color = colors.onSurfaceVariant,
        )
        if (!plan.allWaitsKnown) {
            Text(
                "Bazı sınır kapıları için bekleme verisi yok; varış daha geç olabilir.",
                style = MaterialTheme.typography.labelMedium,
                color = LocalLevelColors.current.medium,
            )
        }
    }
}

@Composable
private fun Alternatives(alternatives: List<TripAlternative>) {
    val colors = MaterialTheme.colorScheme
    Column(Modifier.fillMaxWidth().clip(MaterialTheme.shapes.large).background(colors.surface).padding(vertical = 8.dp)) {
        Text(
            "Güzergâh seçenekleri",
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp),
        )
        alternatives.forEachIndexed { i, alt ->
            if (i > 0) HorizontalDivider(Modifier.padding(horizontal = 20.dp), color = colors.outlineVariant)
            Row(Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 12.dp), verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text(
                        alt.via.ifEmpty { listOf("Sınır geçişi yok") }.joinToString(" · "),
                        style = MaterialTheme.typography.bodyMedium,
                        color = colors.onSurface,
                    )
                    Text(
                        "${formatNumber(alt.distanceKm.roundToInt())} km" +
                            if (alt.allWaitsKnown) " · sınır ${formatDuration(alt.borderWaitMin)}" else " · bekleme verisi yok",
                        style = MaterialTheme.typography.labelMedium,
                        color = if (alt.allWaitsKnown) colors.onSurfaceVariant else LocalLevelColors.current.medium,
                    )
                }
                Text(
                    formatDuration(alt.totalMin) + if (alt.allWaitsKnown) "" else "+",
                    style = MaterialTheme.typography.titleMedium,
                    color = if (alt.selected) colors.onSurface else colors.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun Timeline(plan: TripPlan) {
    val colors = MaterialTheme.colorScheme
    val names = plan.crossings.associate { it.id to it.name }
    Column(Modifier.fillMaxWidth().clip(MaterialTheme.shapes.large).background(colors.surface).padding(20.dp)) {
        Text("Sefer planı", style = MaterialTheme.typography.titleMedium)
        Spacer(Modifier.height(16.dp))
        plan.steps.forEach { step ->
            val (icon, tint) = stepIcon(step.kind)
            TimelineRow(
                icon = icon,
                tint = tint,
                title = stepTitle(step, names),
                subtitle = stepSubtitle(step),
                time = formatClock(step.start),
                isLast = false,
            )
        }
        TimelineRow(
            icon = Icons.Rounded.Place,
            tint = colors.onSurface,
            title = "Varış",
            subtitle = "${formatNumber(plan.distanceKm.roundToInt())} km",
            time = formatClock(plan.arrival),
            isLast = true,
        )
    }
}

@Composable
private fun TimelineRow(icon: ImageVector, tint: Color, title: String, subtitle: String, time: String, isLast: Boolean) {
    val colors = MaterialTheme.colorScheme
    Row(Modifier.fillMaxWidth().height(IntrinsicSize.Min)) {
        Text(
            time,
            style = MaterialTheme.typography.labelMedium,
            color = colors.onSurfaceVariant,
            modifier = Modifier.width(72.dp).padding(top = 8.dp),
        )
        Column(Modifier.width(36.dp).fillMaxHeight(), horizontalAlignment = Alignment.CenterHorizontally) {
            Box(
                Modifier.size(32.dp).clip(CircleShape).background(tint.copy(alpha = 0.12f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(16.dp))
            }
            if (!isLast) {
                Box(Modifier.width(2.dp).weight(1f).background(colors.outlineVariant))
            }
        }
        Column(Modifier.weight(1f).padding(start = 12.dp, top = 6.dp, bottom = if (isLast) 0.dp else 18.dp)) {
            Text(title, style = MaterialTheme.typography.bodyMedium, color = colors.onSurface)
            Text(subtitle, style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
        }
    }
}

@Composable
private fun stepIcon(kind: StepKind): Pair<ImageVector, Color> {
    val levels = LocalLevelColors.current
    return when (kind) {
        StepKind.DRIVE -> Icons.Rounded.LocalShipping to MaterialTheme.colorScheme.onSurface
        StepKind.BREAK -> Icons.Rounded.LocalCafe to levels.medium
        StepKind.DAILY_REST -> Icons.Rounded.Hotel to Color(0xFF5B6CFF)
        StepKind.BORDER_WAIT -> Icons.Rounded.Flag to levels.high
    }
}

private fun stepTitle(step: TripStep, crossingNames: Map<String, String>): String = when (step.kind) {
    StepKind.DRIVE -> "Sürüş · ${formatDuration(step.durationMin)}"
    StepKind.BREAK -> "${step.durationMin} dk mola"
    StepKind.DAILY_REST -> "${formatDuration(step.durationMin)} günlük dinlenme"
    StepKind.BORDER_WAIT -> "${crossingNames[step.crossingId] ?: "Sınır"} · ${formatDuration(step.durationMin)} bekleme"
}

private fun stepSubtitle(step: TripStep): String = when (step.kind) {
    StepKind.DRIVE -> "km ${step.fromKm.roundToInt()} → ${step.toKm.roundToInt()} · ${(step.toKm - step.fromKm).roundToInt()} km"
    StepKind.BREAK, StepKind.DAILY_REST -> "km ${step.fromKm.roundToInt()} · " + when (step.reason) {
        StopReason.CONTINUOUS_DRIVING -> "4,5 saat kesintisiz sürüş doldu"
        StopReason.DAILY_DRIVING -> "Günlük sürüş süresi doldu"
        StopReason.DUTY_PERIOD -> "13 saatlik görev süresi doldu"
        null -> "Zorunlu mola"
    }
    StepKind.BORDER_WAIT -> "km ${step.fromKm.roundToInt()} · " +
        if (step.durationMin >= 11 * 60) "günlük dinlenme yerine sayılır" else if (step.durationMin >= 45) "mola yerine sayılır" else "tahmini bekleme"
}
