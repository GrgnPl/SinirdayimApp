package com.sinirdayim.data.remote

import kotlin.test.Test
import kotlin.test.assertEquals

class PolylineTest {
    @Test
    fun decodesGoogleReferenceExample() {
        // https://developers.google.com/maps/documentation/utilities/polylinealgorithm
        val points = decodePolyline("_p~iF~ps|U_ulLnnqC_mqNvxq`@", precision = 5)
        assertEquals(3, points.size)
        assertEquals(38.5, points[0].lat, 1e-9)
        assertEquals(-120.2, points[0].lng, 1e-9)
        assertEquals(40.7, points[1].lat, 1e-9)
        assertEquals(-120.95, points[1].lng, 1e-9)
        assertEquals(43.252, points[2].lat, 1e-9)
        assertEquals(-126.453, points[2].lng, 1e-9)
    }

    @Test
    fun emptyInputGivesNoPoints() {
        assertEquals(0, decodePolyline("").size)
    }
}
