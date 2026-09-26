package com.sinirdayim.data.remote

import com.sinirdayim.domain.model.GeoPoint

/** Decodes an encoded polyline; Valhalla uses precision 6. */
fun decodePolyline(encoded: String, precision: Int = 6): List<GeoPoint> {
    var factor = 1.0
    repeat(precision) { factor *= 10 }
    val points = ArrayList<GeoPoint>()
    var index = 0
    var lat = 0
    var lng = 0
    while (index < encoded.length) {
        val values = IntArray(2)
        for (k in 0..1) {
            var shift = 0
            var result = 0
            while (true) {
                if (index >= encoded.length) return points // malformed tail
                val b = encoded[index++].code - 63
                result = result or ((b and 0x1f) shl shift)
                shift += 5
                if (b < 0x20) break
            }
            values[k] = if (result and 1 != 0) (result shr 1).inv() else result shr 1
        }
        lat += values[0]
        lng += values[1]
        points += GeoPoint(lat / factor, lng / factor)
    }
    return points
}
