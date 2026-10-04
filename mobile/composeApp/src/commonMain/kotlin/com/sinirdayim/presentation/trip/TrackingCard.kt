@file:OptIn(ExperimentalTime::class)

package com.sinirdayim.presentation.trip

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.sinirdayim.domain.model.GeoPoint
import com.sinirdayim.domain.usecase.TripProgress
import com.sinirdayim.presentation.theme.LocalLevelColors
import com.sinirdayim.presentation.util.formatClock
import com.sinirdayim.presentation.util.formatDuration
import org.maplibre.compose.location.LocationAccuracy
import org.maplibre.compose.location.LocationEvent
import org.maplibre.compose.location.LocationPermission
import org.maplibre.compose.location.LocationRequest
import org.maplibre.compose.location.LocationUnavailableReason
import org.maplibre.compose.location.rememberDefaultLocationProvider
import org.maplibre.spatialk.units.extensions.meters
import kotlin.math.roundToInt
import kotlin.time.Duration
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Duration.Companion.seconds
import kotlin.time.ExperimentalTime

/**
 * Follows the trip with GPS while the app is open: progress against the
 * plan and whether the booked border slot is still reachable.
 */
@Composable
fun TrackingCard(state: TripUiState, viewModel: TripViewModel, rssUrl: String?) {
    val colors = MaterialTheme.colorScheme
    val provider = rememberDefaultLocationProvider()
    val permission by provider.permission.collectAsState()

    LaunchedEffect(state.tracking, permission) {
        if (!state.tracking) return@LaunchedEffect
        if (permission !is LocationPermission.Granted) {
            provider.requestPermission()
            return@LaunchedEffect
        }
        provider.updates(LocationRequest(accuracy = LocationAccuracy.High, minimumInterval = 10.seconds, minimumDistance = 50.meters))
            .collect { event ->
                when (event) {
                    is LocationEvent.Update -> {
                        val p = event.measurement.position
                        viewModel.onLocation(GeoPoint(p.latitude, p.longitude), event.measurement.measuredAt)
                    }
                    is LocationEvent.Unavailable -> viewModel.onLocationProblem(event.reason.message())
                }
            }
    }

    Column(
        Modifier.fillMaxWidth().clip(MaterialTheme.shapes.large).background(colors.surface).padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text("Yolculuk takibi", style = MaterialTheme.typography.titleMedium)
        if (!state.tracking) {
            Text(
                "Yola çıkınca başlat: konumuna göre planın gerisinde mi ilerisinde mi olduğunu ve sınır randevuna yetişip yetişmeyeceğini gösterir. Uygulama açıkken çalışır.",
                style = MaterialTheme.typography.labelMedium,
                color = colors.onSurfaceVariant,
            )
            Button(
                onClick = viewModel::startTracking,
                shape = RoundedCornerShape(12.dp),
                colors = ButtonDefaults.buttonColors(containerColor = colors.primary, contentColor = colors.onPrimary),
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Yolculuğu başlat") }
            return@Column
        }

        val notGranted = permission as? LocationPermission.NotGranted
        when {
            notGranted != null -> Text(
                if (notGranted.canRequest == false) "Konum izni kapalı. Ayarlardan izin verince takip başlar." else "Takip için konum izni gerekiyor.",
                style = MaterialTheme.typography.bodyMedium,
                color = LocalLevelColors.current.medium,
            )
            state.locationProblem != null -> Text(state.locationProblem, style = MaterialTheme.typography.bodyMedium, color = LocalLevelColors.current.medium)
            state.progress == null -> Text("Konum bekleniyor…", style = MaterialTheme.typography.bodyMedium, color = colors.onSurfaceVariant)
            else -> ProgressDetails(state.progress, rssUrl)
        }
        TextButton(onClick = viewModel::stopTracking) { Text("Takibi durdur") }
    }
}

@Composable
private fun ProgressDetails(p: TripProgress, rssUrl: String?) {
    val colors = MaterialTheme.colorScheme
    val levels = LocalLevelColors.current
    val uriHandler = LocalUriHandler.current

    if (p.offRoute) {
        Text(
            "Rotadan ${p.offRouteKm.roundToInt()} km uzaktasın. Plan rotaya dönünce yeniden takip edilir.",
            style = MaterialTheme.typography.bodyMedium,
            color = levels.medium,
        )
        return
    }
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Row {
            Text("km ${p.km.roundToInt()} / ${p.totalKm.roundToInt()}", style = MaterialTheme.typography.bodyMedium, modifier = Modifier.weight(1f))
            Text(delayText(p.delay), style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Medium, color = delayColor(p.delay))
        }
        LinearProgressIndicator(
            progress = { if (p.totalKm > 0) (p.km / p.totalKm).toFloat().coerceIn(0f, 1f) else 0f },
            modifier = Modifier.fillMaxWidth().height(6.dp).clip(RoundedCornerShape(3.dp)),
            color = colors.primary,
            trackColor = colors.surfaceVariant,
        )
        Text("Tahmini varış ${formatClock(p.projectedArrival)}", style = MaterialTheme.typography.labelMedium, color = colors.onSurfaceVariant)
    }

    val a = p.appointment ?: return
    if (a.passed) {
        Text("${a.crossingName} geçildi.", style = MaterialTheme.typography.bodyMedium, color = levels.low)
        return
    }
    val late = a.lateBy.isPositive()
    Box(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp))
            .background((if (late) levels.high else levels.low).copy(alpha = 0.10f)).padding(12.dp),
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(
                if (late) "Randevuna ${formatDuration(a.lateBy.inWholeMinutes.toInt())} geç kalacaksın"
                else "Randevuna yetişiyorsun",
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.SemiBold,
                color = if (late) levels.high else levels.low,
            )
            Text(
                "${a.crossingName}: tahmini varış ${formatClock(a.projectedGateArrival)}, randevu ${formatClock(a.slot)}",
                style = MaterialTheme.typography.labelMedium,
            )
            if (late) {
                Text("RSS'ten randevunu ertele, sonra yeni saati burada güncelle.", style = MaterialTheme.typography.labelMedium)
                if (rssUrl != null) {
                    OutlinedButton(onClick = { uriHandler.openUri(rssUrl) }, shape = RoundedCornerShape(12.dp)) { Text("RSS'i aç") }
                }
            }
        }
    }
}

private val onTimeTolerance = 5.minutes

private fun delayText(d: Duration): String = when {
    d > onTimeTolerance -> "planın ${formatDuration(d.inWholeMinutes.toInt())} gerisinde"
    d < -onTimeTolerance -> "planın ${formatDuration((-d).inWholeMinutes.toInt())} ilerisinde"
    else -> "plana uygun"
}

@Composable
private fun delayColor(d: Duration) = when {
    d > onTimeTolerance -> LocalLevelColors.current.medium
    d < -onTimeTolerance -> LocalLevelColors.current.low
    else -> MaterialTheme.colorScheme.onSurface
}

private fun LocationUnavailableReason.message(): String = when (this) {
    LocationUnavailableReason.ServicesDisabled -> "Konum servisleri kapalı."
    LocationUnavailableReason.PermissionDenied -> "Konum izni verilmedi."
    LocationUnavailableReason.TemporarilyUnavailable -> "Konum şu an alınamıyor, tekrar denenecek."
    LocationUnavailableReason.Unsupported -> "Bu cihaz konum desteklemiyor."
    LocationUnavailableReason.UnexpectedFailure -> "Konum alınırken bir hata oluştu."
}
