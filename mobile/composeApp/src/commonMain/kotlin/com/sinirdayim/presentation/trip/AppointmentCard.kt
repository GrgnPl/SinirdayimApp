@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.presentation.trip

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.EventAvailable
import androidx.compose.material.icons.rounded.LocalParking
import androidx.compose.material.icons.rounded.Remove
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.sinirdayim.domain.model.AppointmentPlan
import com.sinirdayim.domain.model.ProcedureKind
import com.sinirdayim.domain.model.TripCrossing
import com.sinirdayim.domain.model.TripPlan
import com.sinirdayim.presentation.theme.LocalLevelColors
import com.sinirdayim.presentation.util.formatClock
import com.sinirdayim.presentation.util.formatDuration
import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.LocalDate
import kotlinx.datetime.LocalDateTime
import kotlinx.datetime.LocalTime
import kotlinx.datetime.TimeZone
import kotlinx.datetime.atTime
import kotlinx.datetime.plus
import kotlinx.datetime.toInstant
import kotlinx.datetime.toLocalDateTime
import kotlin.math.abs
import kotlin.time.Clock
import kotlin.time.ExperimentalTime
import kotlin.time.Instant

/**
 * Border formalities on the route: RSS appointments (suggest, book, plan
 * around the slot) and mandatory truck parks. Shows nothing when none apply.
 */
@Composable
fun ProceduresCard(
    plan: TripPlan,
    isPlanning: Boolean,
    error: String?,
    onSetAppointment: (crossingId: String, at: Instant) -> Unit,
    onClearAppointment: () -> Unit,
) {
    val crossings = plan.crossings.filter { it.procedures.isNotEmpty() }
    if (crossings.isEmpty()) return
    val colors = MaterialTheme.colorScheme
    var pickerFor by remember { mutableStateOf<TripCrossing?>(null) }

    Column(
        Modifier.fillMaxWidth().clip(MaterialTheme.shapes.large).background(colors.surface).padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Text("Sınır işlemleri", style = MaterialTheme.typography.titleMedium)
        crossings.forEachIndexed { i, c ->
            if (i > 0) HorizontalDivider(color = colors.outlineVariant)
            c.appointmentSystem?.let { system ->
                val booked = plan.appointment?.takeIf { it.crossingId == c.id }
                if (booked != null) {
                    BookedAppointment(c, booked, isPlanning, onChange = { pickerFor = c }, onClear = onClearAppointment)
                } else {
                    AppointmentSuggestion(c, system.url, isPlanning, onEnter = { pickerFor = c })
                }
            }
            c.procedures.filter { it.kind == ProcedureKind.TRUCK_PARK }.forEach { p ->
                InfoRow(
                    Icons.Rounded.LocalParking,
                    "${c.name}: ${p.system}",
                    listOfNotNull(if (p.mandatory) "Zorunlu" else null, p.fee, p.note).joinToString(" · "),
                )
            }
        }
        error?.let { Text(it, style = MaterialTheme.typography.labelMedium, color = colors.error) }
    }

    pickerFor?.let { c ->
        val initial = plan.appointment?.takeIf { it.crossingId == c.id }?.at
            ?: c.suggestedAppointment
            ?: Clock.System.now()
        AppointmentPicker(
            title = "${c.name} randevun",
            initial = initial,
            onConfirm = { at ->
                pickerFor = null
                onSetAppointment(c.id, at)
            },
            onDismiss = { pickerFor = null },
        )
    }
}

@Composable
private fun AppointmentSuggestion(c: TripCrossing, url: String?, isPlanning: Boolean, onEnter: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    val uriHandler = LocalUriHandler.current
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        InfoRow(
            Icons.Rounded.EventAvailable,
            "${c.name}: RSS randevusu zorunlu",
            "Randevusuz çıkış yapılamaz. Randevu, çıkış beyanındaki gümrüğe bağlıdır.",
        )
        c.suggestedAppointment?.let {
            Row(verticalAlignment = Alignment.Bottom) {
                Text("Önerilen randevu", style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant, modifier = Modifier.weight(1f))
                Text(formatClock(it), style = MaterialTheme.typography.headlineSmall)
            }
            Text(
                "Plana göre kapıya bu saatte varıyorsun. RSS'te bu saate en yakın boş dilimi al, sonra buraya gir; planı randevuna göre yeniden kuralım.",
                style = MaterialTheme.typography.labelMedium,
                color = colors.onSurfaceVariant,
            )
        }
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            if (url != null) {
                OutlinedButton(onClick = { uriHandler.openUri(url) }, shape = RoundedCornerShape(12.dp), modifier = Modifier.weight(1f)) {
                    Text("RSS'i aç")
                }
            }
            Button(
                onClick = onEnter,
                enabled = !isPlanning,
                shape = RoundedCornerShape(12.dp),
                colors = ButtonDefaults.buttonColors(containerColor = colors.primary, contentColor = colors.onPrimary),
                modifier = Modifier.weight(1f),
            ) {
                Text("Randevumu gir")
            }
        }
    }
}

@Composable
private fun BookedAppointment(c: TripCrossing, a: AppointmentPlan, isPlanning: Boolean, onChange: () -> Unit, onClear: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    val levels = LocalLevelColors.current
    val statusColor = if (a.onTime) levels.low else levels.high
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text("${c.name} · RSS randevun", style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
                Text(formatClock(a.at), style = MaterialTheme.typography.headlineSmall)
            }
            Box(Modifier.clip(CircleShape).background(statusColor.copy(alpha = 0.12f)).padding(horizontal = 10.dp, vertical = 4.dp)) {
                Text(
                    if (a.onTime) "Yetişiyorsun" else "Geç kalıyorsun",
                    style = MaterialTheme.typography.labelMedium,
                    color = statusColor,
                )
            }
        }
        Text(
            "Kapıya varış ${formatClock(a.arriveAt)} · " +
                if (a.slackMin >= 0) "${formatDuration(a.slackMin)} erken" else "${formatDuration(abs(a.slackMin))} geç",
            style = MaterialTheme.typography.bodyMedium,
        )
        val latest = a.latestDeparture
        Text(
            when {
                latest == null -> "Bu randevuya yetişmek mümkün görünmüyor; RSS'ten ertele."
                latest < Clock.System.now() -> "En geç kalkış (${formatClock(latest)}) geçti; RSS'ten ertelemeyi düşün."
                else -> "En geç kalkış: ${formatClock(latest)} (kapıda 30 dk pay ile)"
            },
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium,
            color = if (latest == null || latest < Clock.System.now()) levels.high else colors.onSurface,
        )
        Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
            TextButton(onClick = onChange, enabled = !isPlanning) { Text("Değiştir") }
            TextButton(onClick = onClear, enabled = !isPlanning) { Text("Randevuyu kaldır") }
        }
    }
}

@Composable
private fun InfoRow(icon: ImageVector, title: String, subtitle: String) {
    val colors = MaterialTheme.colorScheme
    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        Box(Modifier.size(32.dp).clip(CircleShape).background(colors.surfaceVariant), contentAlignment = Alignment.Center) {
            Icon(icon, contentDescription = null, modifier = Modifier.size(16.dp))
        }
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyMedium, color = colors.onSurface)
            if (subtitle.isNotBlank()) {
                Text(subtitle, style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
            }
        }
    }
}

private val weekdays = listOf("Pzt", "Sal", "Çar", "Per", "Cum", "Cmt", "Paz")

/** Date chips for a week plus a half-hour time stepper; keeps the form short. */
@Composable
private fun AppointmentPicker(title: String, initial: Instant, onConfirm: (Instant) -> Unit, onDismiss: () -> Unit) {
    val tz = TimeZone.currentSystemDefault()
    val start = initial.toLocalDateTime(tz)
    val today = Clock.System.now().toLocalDateTime(tz).date
    val firstDay = minOf(today, start.date)
    var date by remember { mutableStateOf(start.date) }
    var minuteOfDay by remember { mutableStateOf((start.hour * 60 + start.minute) / 30 * 30) }
    val colors = MaterialTheme.colorScheme

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    (0 until 8).map { firstDay.plus(it, DateTimeUnit.DAY) }.forEach { d ->
                        DayChip(d, selected = d == date) { date = d }
                    }
                }
                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.Center) {
                    StepButton(Icons.Rounded.Remove, "30 dk önce") { minuteOfDay = (minuteOfDay - 30).mod(24 * 60) }
                    Text(
                        "${(minuteOfDay / 60).toString().padStart(2, '0')}:${(minuteOfDay % 60).toString().padStart(2, '0')}",
                        style = MaterialTheme.typography.displayLarge,
                        color = colors.onSurface,
                        modifier = Modifier.padding(horizontal = 20.dp),
                    )
                    StepButton(Icons.Rounded.Add, "30 dk sonra") { minuteOfDay = (minuteOfDay + 30).mod(24 * 60) }
                }
                Text(
                    "RSS'te aldığın randevunun tarih ve saatini seç.",
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.onSurfaceVariant,
                )
            }
        },
        confirmButton = {
            TextButton(onClick = {
                onConfirm(date.atTime(LocalTime(minuteOfDay / 60, minuteOfDay % 60)).toInstant(tz))
            }) { Text("Planla") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Vazgeç") } },
    )
}

@Composable
private fun DayChip(d: LocalDate, selected: Boolean, onClick: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    Column(
        Modifier
            .clip(RoundedCornerShape(12.dp))
            .background(if (selected) colors.primary else colors.surfaceVariant)
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        val fg = if (selected) colors.onPrimary else colors.onSurface
        Text(weekdays[d.dayOfWeek.ordinal], style = MaterialTheme.typography.labelMedium, color = fg)
        Text(d.day.toString(), style = MaterialTheme.typography.titleMedium, color = fg)
    }
}

@Composable
private fun StepButton(icon: ImageVector, description: String, onClick: () -> Unit) {
    Box(
        Modifier.size(40.dp).clip(CircleShape).background(MaterialTheme.colorScheme.surfaceVariant).clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(icon, contentDescription = description, modifier = Modifier.size(18.dp))
    }
}
