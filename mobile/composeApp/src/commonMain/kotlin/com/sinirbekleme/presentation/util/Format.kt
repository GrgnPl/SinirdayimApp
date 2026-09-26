@file:OptIn(ExperimentalTime::class)

package com.sinirbekleme.presentation.util

import com.sinirbekleme.domain.model.Direction
import com.sinirbekleme.domain.model.Level
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime
import kotlin.math.roundToInt
import kotlin.time.Clock
import kotlin.time.ExperimentalTime
import kotlin.time.Instant

/** 45 → "45 dk", 320 → "5 sa 20 dk", 1973 → "1 g 9 sa" */
fun formatDuration(minutes: Int): String {
    val d = minutes / (60 * 24)
    val h = (minutes / 60) % 24
    val m = minutes % 60
    return when {
        d > 0 -> if (h > 0) "$d g $h sa" else "$d g"
        h > 0 -> if (m > 0) "$h sa $m dk" else "$h sa"
        else -> "$m dk"
    }
}

/** Short form for large display: "33 sa", "45 dk", "2 g". */
fun formatDurationShort(minutes: Int): Pair<String, String> = when {
    minutes >= 48 * 60 -> ((minutes / 60f / 24f).roundToInt().toString()) to "gün"
    minutes >= 60 -> ((minutes / 60f).roundToInt().toString()) to "saat"
    else -> minutes.toString() to "dk"
}

fun formatRelative(at: Instant, now: Instant = Clock.System.now()): String {
    val minutes = (now - at).inWholeMinutes
    return when {
        minutes < 1 -> "az önce"
        minutes < 60 -> "$minutes dk önce"
        minutes < 60 * 24 -> "${minutes / 60} sa önce"
        else -> "${minutes / (60 * 24)} gün önce"
    }
}

fun formatNumber(n: Int): String =
    n.toString().reversed().chunked(3).joinToString(".").reversed()

fun formatKm(km: Double): String =
    if (km % 1.0 == 0.0) km.toInt().toString() else ((km * 10).roundToInt() / 10.0).toString().replace('.', ',')

fun Level.label(): String = when (this) {
    Level.LOW -> "Akıcı"
    Level.MEDIUM -> "Yoğun"
    Level.HIGH -> "Çok yoğun"
    Level.UNKNOWN -> "Veri yok"
}

fun Direction.label(): String = when (this) {
    Direction.EXPORT -> "Çıkış"
    Direction.IMPORT -> "Giriş"
}

/** "TR" → 🇹🇷 using regional indicator symbols. */
fun flagEmoji(countryCode: String): String {
    if (countryCode.length != 2) return ""
    val base = 0x1F1E6 - 'A'.code
    return countryCode.uppercase().map { ch ->
        val cp = base + ch.code
        // Build the surrogate pair manually; common code cannot use Character.toChars.
        val v = cp - 0x10000
        charArrayOf((0xD800 + (v shr 10)).toChar(), (0xDC00 + (v and 0x3FF)).toChar()).concatToString()
    }.joinToString("")
}

private val weekdaysTr = listOf("Pzt", "Sal", "Çar", "Per", "Cum", "Cmt", "Paz")

/** "Çar 04:52" in the device time zone. */
fun formatClock(at: Instant): String {
    val t = at.toLocalDateTime(TimeZone.currentSystemDefault())
    val day = weekdaysTr[t.dayOfWeek.ordinal]
    return "$day ${t.hour.toString().padStart(2, '0')}:${t.minute.toString().padStart(2, '0')}"
}
