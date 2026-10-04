package com.sinirdayim.presentation.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.sinirdayim.domain.model.Level

/** Traffic level colors, tuned for both themes. */
@Immutable
data class LevelColors(
    val low: Color,
    val medium: Color,
    val high: Color,
    val unknown: Color,
) {
    fun of(level: Level) = when (level) {
        Level.LOW -> low
        Level.MEDIUM -> medium
        Level.HIGH -> high
        Level.UNKNOWN -> unknown
    }
}

private val LightLevels = LevelColors(
    low = Color(0xFF1F9D63),
    medium = Color(0xFFC98A00),
    high = Color(0xFFD5433A),
    unknown = Color(0xFF9AA0A6),
)

private val DarkLevels = LevelColors(
    low = Color(0xFF4CC98E),
    medium = Color(0xFFF2B53A),
    high = Color(0xFFFF6B5E),
    unknown = Color(0xFF7C828A),
)

val LocalLevelColors = staticCompositionLocalOf { LightLevels }

private val LightScheme = lightColorScheme(
    primary = Color(0xFF1B1D21),
    onPrimary = Color.White,
    background = Color(0xFFF6F6F3),
    onBackground = Color(0xFF15171A),
    surface = Color.White,
    onSurface = Color(0xFF15171A),
    surfaceVariant = Color(0xFFEDEDE9),
    onSurfaceVariant = Color(0xFF6B7078),
    outlineVariant = Color(0xFFE4E4DF),
)

private val DarkScheme = darkColorScheme(
    primary = Color(0xFFF2F2EF),
    onPrimary = Color(0xFF15171A),
    background = Color(0xFF0E1013),
    onBackground = Color(0xFFEDEDEA),
    surface = Color(0xFF17191D),
    onSurface = Color(0xFFEDEDEA),
    surfaceVariant = Color(0xFF22252A),
    onSurfaceVariant = Color(0xFF9CA1A8),
    outlineVariant = Color(0xFF2A2D32),
)

private val AppTypography = Typography().run {
    copy(
        displayLarge = TextStyle(fontSize = 56.sp, lineHeight = 60.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-1.5).sp),
        headlineLarge = TextStyle(fontSize = 32.sp, lineHeight = 38.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-0.5).sp),
        headlineSmall = TextStyle(fontSize = 24.sp, lineHeight = 28.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-0.3).sp),
        titleMedium = TextStyle(fontSize = 16.sp, lineHeight = 22.sp, fontWeight = FontWeight.SemiBold),
        bodyMedium = TextStyle(fontSize = 14.sp, lineHeight = 20.sp),
        labelMedium = TextStyle(fontSize = 12.sp, lineHeight = 16.sp, fontWeight = FontWeight.Medium, letterSpacing = 0.2.sp),
    )
}

private val AppShapes = Shapes(
    small = RoundedCornerShape(10.dp),
    medium = RoundedCornerShape(16.dp),
    large = RoundedCornerShape(22.dp),
)

@Composable
fun AppTheme(dark: Boolean = isSystemInDarkTheme(), content: @Composable () -> Unit) {
    androidx.compose.runtime.CompositionLocalProvider(
        LocalLevelColors provides if (dark) DarkLevels else LightLevels,
    ) {
        MaterialTheme(
            colorScheme = if (dark) DarkScheme else LightScheme,
            typography = AppTypography,
            shapes = AppShapes,
            content = content,
        )
    }
}
