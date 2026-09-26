package com.sinirbekleme.data.remote

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.plugins.timeout
import io.ktor.client.request.get
import io.ktor.client.request.parameter
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.http.ContentType
import io.ktor.http.contentType

class TripApi(private val client: HttpClient) {
    suspend fun places(query: String): PlaceListDto =
        client.get("v1/places") { parameter("q", query) }.body()

    suspend fun plan(request: PlanRequestDto): TripPlanDto =
        client.post("v1/trips/plan") {
            // Planning routes through several candidate crossings.
            timeout { requestTimeoutMillis = 120_000 }
            contentType(ContentType.Application.Json)
            setBody(request)
        }.body()
}
