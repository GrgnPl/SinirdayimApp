package com.sinirdayim.data.remote

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.defaultRequest
import io.ktor.client.request.get
import io.ktor.client.request.parameter
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.http.ContentType
import io.ktor.http.contentType
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json

class CrossingApi(private val client: HttpClient) {
    suspend fun crossings(): CrossingListDto = client.get("v1/crossings").body()

    suspend fun crossing(id: String, hours: Int): CrossingDetailDto =
        client.get("v1/crossings/$id") { parameter("hours", hours) }.body()

    suspend fun queue(id: String): QueueDetailDto = client.get("v1/crossings/$id/queue").body()

    suspend fun report(id: String, report: DriverReportDto) {
        client.post("v1/crossings/$id/reports") {
            contentType(ContentType.Application.Json)
            setBody(report)
        }
    }

    companion object {
        fun createClient(baseUrl: String): HttpClient = HttpClient {
            expectSuccess = true
            install(ContentNegotiation) {
                json(Json { ignoreUnknownKeys = true })
            }
            install(HttpTimeout) {
                requestTimeoutMillis = 15_000
            }
            defaultRequest { url(baseUrl.trimEnd('/') + "/") }
        }
    }
}
