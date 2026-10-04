@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.presentation.detail

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.TrendingDown
import androidx.compose.material.icons.automirrored.rounded.TrendingFlat
import androidx.compose.material.icons.automirrored.rounded.TrendingUp
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Remove
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.sinirdayim.domain.model.Direction
import com.sinirdayim.domain.model.DirectionQueue
import com.sinirdayim.domain.model.DriverReport
import com.sinirdayim.domain.model.NewReport
import com.sinirdayim.domain.model.ReportKind
import com.sinirdayim.domain.model.TrendDirection
import com.sinirdayim.presentation.components.DirectionToggle
import com.sinirdayim.presentation.theme.LocalLevelColors
import com.sinirdayim.presentation.util.formatDuration
import com.sinirdayim.presentation.util.formatNumber
import com.sinirdayim.presentation.util.formatRelative
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime
import kotlin.math.abs
import kotlin.math.roundToInt
import kotlin.time.ExperimentalTime

@Composable
private fun Card(content: @Composable () -> Unit) {
    Column(
        Modifier.fillMaxWidth().clip(MaterialTheme.shapes.large).background(MaterialTheme.colorScheme.surface).padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) { content() }
}

/** "If you reach the gate at HH:00, you wait ..." as bars for the next hours. */
@Composable
fun OutlookCard(q: DirectionQueue) {
    if (q.outlook.isEmpty()) return
    val colors = MaterialTheme.colorScheme
    val levels = LocalLevelColors.current
    val maxWait = q.outlook.maxOf { it.waitMinutes }.coerceAtLeast(60)
    val best = q.outlook.minBy { it.waitMinutes }
    val tz = TimeZone.currentSystemDefault()
    Card {
        Row(verticalAlignment = Alignment.Bottom) {
            Text("Saat saat bekleme", style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f))
            Text("kapıya varış saatine göre", style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
        }
        Row(Modifier.fillMaxWidth().height(120.dp), horizontalArrangement = Arrangement.spacedBy(4.dp), verticalAlignment = Alignment.Bottom) {
            q.outlook.forEach { o ->
                val frac = (o.waitMinutes.toFloat() / maxWait).coerceIn(0.03f, 1f)
                Box(Modifier.weight(1f).fillMaxHeight(), contentAlignment = Alignment.BottomCenter) {
                    Box(
                        Modifier.fillMaxWidth().fillMaxHeight(frac)
                            .clip(RoundedCornerShape(topStart = 4.dp, topEnd = 4.dp))
                            .background(levels.of(o.level).copy(alpha = if (o == best) 1f else 0.55f)),
                    )
                }
            }
        }
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(4.dp)) {
            q.outlook.forEachIndexed { i, o ->
                Text(
                    if (i % 3 == 0) o.at.toLocalDateTime(tz).hour.toString().padStart(2, '0') else "",
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.onSurfaceVariant,
                    modifier = Modifier.weight(1f),
                )
            }
        }
        val now = q.outlook.first()
        Text(
            buildString {
                append("Şimdi: ~${formatNumber(now.vehicles)} tır, ${formatDuration(now.waitMinutes)}")
                if (best.waitMinutes < now.waitMinutes - 30) {
                    val h = best.at.toLocalDateTime(tz).hour.toString().padStart(2, '0')
                    append(" · En kısa: $h:00'de ${formatDuration(best.waitMinutes)}")
                }
            },
            style = MaterialTheme.typography.bodyMedium,
        )
        Text(
            "Tahmin: mevcut kuyruk, son ölçülen trend (en fazla 6 saat ileri) ve günlük geçiş hızından hesaplanır.",
            style = MaterialTheme.typography.labelMedium,
            color = colors.onSurfaceVariant,
        )
    }
}

/** Trend and where the numbers come from, with their age. */
@Composable
fun SourcesCard(q: DirectionQueue) {
    val colors = MaterialTheme.colorScheme
    val levels = LocalLevelColors.current
    Card {
        Text("Sıra durumu", style = MaterialTheme.typography.titleMedium)
        q.trend?.let { t ->
            val (icon, color, text) = when (t.direction) {
                TrendDirection.GROWING -> Triple(Icons.AutoMirrored.Rounded.TrendingUp, levels.high, "Kuyruk büyüyor")
                TrendDirection.SHRINKING -> Triple(Icons.AutoMirrored.Rounded.TrendingDown, levels.low, "Kuyruk azalıyor")
                TrendDirection.STABLE -> Triple(Icons.AutoMirrored.Rounded.TrendingFlat, colors.onSurfaceVariant, "Kuyruk sabit")
            }
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Icon(icon, contentDescription = null, tint = color, modifier = Modifier.size(22.dp))
                Column {
                    Text(text, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.SemiBold, color = color)
                    if (t.direction != TrendDirection.STABLE) {
                        Text(
                            "saatte ~${abs(t.vehiclesPerHour).roundToInt()} tır ${if (t.vehiclesPerHour > 0) "artış" else "azalış"}",
                            style = MaterialTheme.typography.labelMedium,
                            color = colors.onSurfaceVariant,
                        )
                    }
                }
            }
        }
        if (q.estimate.method != q.official.method && q.official.waitMinutes != null) {
            Text(
                "Resmi veriye göre ${formatDuration(q.official.waitMinutes)}; şoför bildirimleriyle güncellendi.",
                style = MaterialTheme.typography.labelMedium,
                color = colors.onSurfaceVariant,
            )
        }
        q.dailyThroughput?.let {
            Text("Son 24 saatte ${formatNumber(it)} tır geçti (saatte ~${(it / 24.0).roundToInt()}).", style = MaterialTheme.typography.bodyMedium)
        }
        if (q.sources.isNotEmpty()) {
            HorizontalDivider(color = colors.outlineVariant)
            q.sources.forEach { s ->
                Row {
                    Text(sourceName(s.source) + if (s.reports > 0) " (${s.reports})" else "", style = MaterialTheme.typography.labelMedium, modifier = Modifier.weight(1f))
                    Text(formatRelative(s.observedAt), style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
                }
            }
        }
    }
}

/** Recent reports from drivers and the button to add one. */
@Composable
fun ReportsCard(q: DirectionQueue, sending: Boolean, message: String?, onReport: (NewReport) -> Unit, onMessageShown: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    var dialog by remember { mutableStateOf(false) }
    Card {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("Şoför bildirimleri", style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f))
            Text("son 24 saat", style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
        }
        if (q.reports.isEmpty()) {
            Text("Henüz bildirim yok. Sıradaysan ya da yeni geçtiysen ilk bildiren sen ol.", style = MaterialTheme.typography.bodyMedium, color = colors.onSurfaceVariant)
        }
        q.reports.take(6).forEach { ReportRow(it) }
        message?.let {
            Text(it, style = MaterialTheme.typography.labelMedium, color = LocalLevelColors.current.low, modifier = Modifier.clickable(onClick = onMessageShown))
        }
        Button(
            onClick = { dialog = true },
            enabled = !sending,
            shape = RoundedCornerShape(12.dp),
            colors = ButtonDefaults.buttonColors(containerColor = colors.primary, contentColor = colors.onPrimary),
            modifier = Modifier.fillMaxWidth(),
        ) { Text(if (sending) "Gönderiliyor…" else "Sıra durumunu bildir") }
    }
    if (dialog) {
        ReportDialog(
            direction = q.direction,
            onSend = { dialog = false; onReport(it) },
            onDismiss = { dialog = false },
        )
    }
}

@Composable
private fun ReportRow(r: DriverReport) {
    val colors = MaterialTheme.colorScheme
    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        Box(Modifier.padding(top = 6.dp).size(8.dp).clip(CircleShape).background(if (r.kind == ReportKind.PASSED) LocalLevelColors.current.low else colors.primary))
        Column(Modifier.weight(1f)) {
            Text(
                when (r.kind) {
                    ReportKind.PASSED -> "Geçti · ${r.waitMinutes?.let(::formatDuration) ?: "?"} bekledi"
                    ReportKind.IN_QUEUE -> "Sırada · " + (r.vehiclesAhead?.let { "önünde ~${formatNumber(it)} tır" } ?: r.queueKm?.let { "kapıya ${it.roundToInt()} km" } ?: "")
                },
                style = MaterialTheme.typography.bodyMedium,
            )
            r.note?.let { Text(it, style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant) }
        }
        Text(formatRelative(r.at), style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
    }
}

@Composable
private fun ReportDialog(direction: Direction, onSend: (NewReport) -> Unit, onDismiss: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    // The two-option toggle is reused: left = in the queue, right = passed.
    var kind by remember { mutableStateOf(Direction.EXPORT) }
    var vehicles by remember { mutableStateOf(100) }
    var waitMin by remember { mutableStateOf(120) }
    var note by remember { mutableStateOf("") }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Sıra durumunu bildir") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                DirectionToggle(selected = kind, onSelect = { kind = it }, labels = { if (it == Direction.EXPORT) "Sıradayım" else "Geçtim" })
                if (kind == Direction.EXPORT) {
                    Stepper("Önümdeki tır", formatNumber(vehicles), onMinus = { vehicles = (vehicles - 10).coerceAtLeast(0) }, onPlus = { vehicles = (vehicles + 10).coerceAtMost(5000) })
                } else {
                    Stepper("Toplam bekleme", formatDuration(waitMin), onMinus = { waitMin = (waitMin - 30).coerceAtLeast(0) }, onPlus = { waitMin = (waitMin + 30).coerceAtMost(7 * 24 * 60) })
                }
                Box(Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(colors.surfaceVariant).padding(12.dp)) {
                    if (note.isEmpty()) Text("Not (isteğe bağlı)", style = MaterialTheme.typography.bodyMedium, color = colors.onSurfaceVariant)
                    BasicTextField(
                        value = note,
                        onValueChange = { note = it.take(280) },
                        textStyle = MaterialTheme.typography.bodyMedium.copy(color = colors.onSurface),
                        cursorBrush = SolidColor(colors.onSurface),
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                onSend(
                    if (kind == Direction.EXPORT) NewReport(direction, ReportKind.IN_QUEUE, vehiclesAhead = vehicles, note = note)
                    else NewReport(direction, ReportKind.PASSED, waitMinutes = waitMin, note = note),
                )
            }) { Text("Gönder") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Vazgeç") } },
    )
}

@Composable
private fun Stepper(label: String, value: String, onMinus: () -> Unit, onPlus: () -> Unit) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(label, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.weight(1f))
        RoundButton(Icons.Rounded.Remove, onMinus)
        Text(value, style = MaterialTheme.typography.titleMedium, modifier = Modifier.width(96.dp), textAlign = androidx.compose.ui.text.style.TextAlign.Center)
        RoundButton(Icons.Rounded.Add, onPlus)
    }
}

@Composable
private fun RoundButton(icon: ImageVector, onClick: () -> Unit) {
    Box(
        Modifier.size(36.dp).clip(CircleShape).background(MaterialTheme.colorScheme.surfaceVariant).clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) { Icon(icon, contentDescription = null, modifier = Modifier.size(18.dp)) }
}

private fun sourceName(id: String) = when (id) {
    "und" -> "UND – Uluslararası Nakliyeciler Derneği"
    "drivers" -> "Şoför bildirimleri"
    else -> id.uppercase()
}
