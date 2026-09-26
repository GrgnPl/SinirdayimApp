package com.sinirdayim.presentation.components

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.unit.dp

/**
 * Minimal area chart. [points] are (x, y) pairs with x already ordered;
 * both axes are normalized to the drawing area, y starts at zero.
 */
@Composable
fun TrendChart(
    points: List<Pair<Double, Double>>,
    color: Color,
    gridColor: Color,
    modifier: Modifier = Modifier,
) {
    Canvas(modifier.fillMaxWidth().height(140.dp)) {
        val gridStroke = 1.dp.toPx()
        for (i in 0..3) {
            val y = size.height * i / 3f
            drawLine(gridColor, Offset(0f, y), Offset(size.width, y), gridStroke)
        }
        if (points.isEmpty()) return@Canvas

        val minX = points.first().first
        val maxX = points.last().first
        val maxY = points.maxOf { it.second }.coerceAtLeast(1.0) * 1.15
        val spanX = (maxX - minX).takeIf { it > 0 } ?: 1.0
        fun pos(p: Pair<Double, Double>) = Offset(
            x = if (points.size == 1) size.width / 2 else ((p.first - minX) / spanX * size.width).toFloat(),
            y = (size.height - p.second / maxY * size.height).toFloat(),
        )

        val offsets = points.map(::pos)
        val line = Path().apply {
            moveTo(offsets.first().x, offsets.first().y)
            offsets.drop(1).forEach { lineTo(it.x, it.y) }
        }
        val area = Path().apply {
            addPath(line)
            lineTo(offsets.last().x, size.height)
            lineTo(offsets.first().x, size.height)
            close()
        }
        drawPath(area, Brush.verticalGradient(listOf(color.copy(alpha = 0.22f), color.copy(alpha = 0f))))
        drawPath(line, color, style = Stroke(width = 2.5.dp.toPx(), cap = StrokeCap.Round, join = StrokeJoin.Round))
        val last = offsets.last()
        drawCircle(color, radius = 4.dp.toPx(), center = last)
        drawCircle(Color.White, radius = 2.dp.toPx(), center = last)
    }
}
