package com.sinirdayim.presentation.trip

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
import androidx.compose.runtime.key
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.sinirdayim.domain.model.AppointmentPlan
import com.sinirdayim.domain.model.RestArea
import com.sinirdayim.domain.model.RestAreaKind
import com.sinirdayim.domain.model.StepKind
import com.sinirdayim.domain.model.StopReason
import com.sinirdayim.domain.model.TripAlternative
import com.sinirdayim.domain.model.TripPlan
import com.sinirdayim.domain.model.TripStep
import com.sinirdayim.presentation.components.StatTile
import com.sinirdayim.presentation.map.RestColor
import com.sinirdayim.presentation.map.TripMap
import com.sinirdayim.presentation.theme.LocalLevelColors
import com.sinirdayim.presentation.util.formatClock
import com.sinirdayim.presentation.util.formatDuration
import com.sinirdayim.presentation.util.formatNumber
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

        key(plan) {
            TripMap(plan, Modifier.fillMaxWidth().height(280.dp).clip(MaterialTheme.shapes.large))
        }

        ArrivalHero(plan)

        ProceduresCard(
            plan = plan,
            isPlanning = state.isPlanning,
            error = state.resultError,
            onSetAppointment = viewModel::setAppointment,
            onClearAppointment = viewModel::clearAppointment,
        )

        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            StatTile("Sürüş", formatDuration(plan.totals.drivingMin), Modifier.weight(1f))
            StatTile("Sınır", formatDuration(plan.totals.borderWaitMin), Modifier.weight(1f))
        }
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            StatTile("Mola", formatDuration(plan.totals.breakMin), Modifier.weight(1f))
            StatTile("Günlük dinlenme", formatDuration(plan.totals.dailyRestMin), Modifier.weight(1f))
        }
        if (plan.totals.weeklyRestMin > 0) {
            StatTile("Haftalık dinlenme", formatDuration(plan.totals.weeklyRestMin), Modifier.fillMaxWidth())
        }

        if (plan.alternatives.size > 1) Alternatives(plan.alternatives)

        Timeline(plan)

        Text(
            "Plan AB 561/2006 ve AETR kurallarına göre hesaplandı: 4,5 saatte 45 dk (veya 15+30) mola, " +
                "günde en fazla 9 saat (haftada 2 gün 10 saat) sürüş, 11 saat (hak varsa 9 saat) günlük dinlenme, " +
                "haftada 56, iki haftada 90 saat sürüş. Sınırdaki bekleme, süresine göre mola veya dinlenme sayılır.",
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
                subtitle = stepSubtitle(step, plan.appointment),
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
        StepKind.DAILY_REST, StepKind.WEEKLY_REST -> Icons.Rounded.Hotel to RestColor
        StepKind.BORDER_WAIT -> Icons.Rounded.Flag to levels.high
    }
}

private fun stepTitle(step: TripStep, crossingNames: Map<String, String>): String = when (step.kind) {
    StepKind.DRIVE -> "Sürüş · ${formatDuration(step.durationMin)}"
    StepKind.BREAK -> "${step.durationMin} dk mola" + (step.restArea?.let { " · ${it.displayName()}" } ?: "")
    StepKind.DAILY_REST -> "${formatDuration(step.durationMin)} dinlenme" + (step.restArea?.let { " · ${it.displayName()}" } ?: "")
    StepKind.WEEKLY_REST -> "Haftalık dinlenme · ${formatDuration(step.durationMin)}" + (step.restArea?.let { " · ${it.displayName()}" } ?: "")
    StepKind.BORDER_WAIT -> (crossingNames[step.crossingId] ?: "Sınır kapısı") +
        if (step.durationMin == 0) " · bekleme verisi yok" else " · ${formatDuration(step.durationMin)} bekleme"
}

private fun stepSubtitle(step: TripStep, appointment: AppointmentPlan?): String = when (step.kind) {
    StepKind.DRIVE -> "km ${step.fromKm.roundToInt()} → ${step.toKm.roundToInt()} · ${(step.toKm - step.fromKm).roundToInt()} km"
    StepKind.BREAK, StepKind.DAILY_REST, StepKind.WEEKLY_REST -> listOfNotNull(
        "km ${step.fromKm.roundToInt()}",
        when {
            !step.reduced -> null
            step.kind == StepKind.BREAK -> "bölünmüş molanın 2. kısmı"
            else -> "kısaltılmış dinlenme"
        },
        step.restArea?.facilities()?.takeIf { it.isNotEmpty() },
        if (step.restArea == null) "yol kenarı – yakında tesis bulunamadı" else null,
        when (step.reason) {
            StopReason.CONTINUOUS_DRIVING -> "4,5 saat kesintisiz sürüş"
            StopReason.DAILY_DRIVING -> "günlük sürüş süresi"
            StopReason.DUTY_PERIOD -> "görev süresi doldu"
            StopReason.WEEKLY_DRIVING -> "haftalık sürüş limiti (56/90 sa) – yeni haftaya kadar"
            StopReason.WEEKLY_REST_DUE -> "6 günlük süre doldu"
            null -> null
        },
    ).joinToString(" · ")
    StepKind.BORDER_WAIT -> "km ${step.fromKm.roundToInt()} · " + when {
        appointment?.crossingId == step.crossingId && appointment != null ->
            "RSS randevusu ${formatClock(appointment.at)} – randevuya kadar bekleme ve geçiş"
        step.durationMin == 0 -> "süre plana eklenmedi, varış daha geç olabilir"
        step.countsAs == StepKind.DAILY_REST -> "günlük dinlenme yerine sayılır"
        step.countsAs == StepKind.BREAK -> "mola yerine sayılır"
        else -> "tahmini bekleme"
    }
}

private fun RestArea.displayName(): String = name ?: when (kind) {
    RestAreaKind.SERVICES -> "Servis alanı"
    RestAreaKind.REST_AREA -> "Dinlenme alanı"
    RestAreaKind.TRUCK_PARKING -> "Tır parkı"
    RestAreaKind.TRUCK_FUEL -> "Akaryakıt"
}

/** Known facilities, e.g. "WC, restoran, güvenlikli". */
private fun RestArea.facilities(): String = listOfNotNull(
    if (name != null) when (kind) {
        RestAreaKind.SERVICES -> "servis alanı"
        RestAreaKind.REST_AREA -> "dinlenme alanı"
        RestAreaKind.TRUCK_PARKING -> "tır parkı"
        RestAreaKind.TRUCK_FUEL -> "akaryakıt"
    } else null,
    "WC".takeIf { toilets == true },
    "duş".takeIf { shower == true },
    "restoran".takeIf { restaurant == true },
    "güvenlikli".takeIf { supervised == true },
    "ücretsiz".takeIf { fee == false },
    "ücretli".takeIf { fee == true },
    hgvCapacity?.let { "$it tır kapasiteli" },
).joinToString(", ")
